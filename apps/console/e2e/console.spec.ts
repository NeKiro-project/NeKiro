import {execFileSync} from 'node:child_process';

import {expect, test, type Locator, type Page} from '@playwright/test';

const providerId = required('VITE_NEKIRO_PROVIDER_ID');
const apiBaseURL = required('VITE_NEKIRO_API_BASE_URL');
const publicAgentOrigin = required('VITE_NEKIRO_PUBLIC_AGENT_ORIGIN');
const ownerToken = required('VITE_NEKIRO_OWNER_TOKEN');
const workspaceId = required('VITE_NEKIRO_DEFAULT_WORKSPACE_ID');
const composeFile = required('NEKIRO_E2E_COMPOSE_FILE');
const composeProject = required('NEKIRO_E2E_COMPOSE_PROJECT');
const gatewayProxyTarget = process.env.NEKIRO_E2E_GATEWAY_PROXY_TARGET;

type AgentFixture = {
  id: string;
  name: string;
  endpoint: string;
  service: string;
  capability: string;
  permissions?: string[];
};

type ReleaseEvidence = {
  releaseId: string;
  cardDigest: string;
};

type BrowserLeakTracker = {
  requestUrls: string[];
  requestBodies: string[];
  consoleMessages: string[];
};

const runtimeA: AgentFixture = {
  id: 'runtime-a',
  name: 'Browser Runtime A',
  endpoint: 'http://runtime-a:8091',
  service: 'runtime-a',
  capability: 'runtime.echo',
  permissions: [],
};

const runtimeB: AgentFixture = {
  id: 'runtime-b',
  name: 'Browser Runtime B',
  endpoint: 'http://runtime-b:8092',
  service: 'runtime-b',
  capability: 'runtime.echo',
  permissions: ['text.read'],
};

type PublicShare = {publicAgentId: string; publicUrl: string};

test.describe.configure({mode: 'serial'});

test('production Console completes trusted publication, invocation, trace, and isolated demos', async ({page}) => {
  const apiRequests: string[] = [];
  const requestUrls: string[] = [];
  const requestBodies: string[] = [];
  const consoleMessages: string[] = [];
  const leakTracker: BrowserLeakTracker = {requestUrls, requestBodies, consoleMessages};
  if (gatewayProxyTarget) {
    const targetOrigin = new URL(gatewayProxyTarget).origin;
    await page.route(`${apiBaseURL}/**`, async (route) => {
      const requestURL = new URL(route.request().url());
      const targetURL = new URL(requestURL.pathname + requestURL.search, targetOrigin + '/');
      const response = await route.fetch({url: targetURL.toString()});
      await route.fulfill({response});
    });
  }
  page.on('request', (request) => {
    requestUrls.push(request.url());
    if (request.postData()) requestBodies.push(request.postData() ?? '');
    if (/\/v[34]\//.test(request.url())) apiRequests.push(request.url());
  });
  page.on('console', (message) => consoleMessages.push(message.text()));

  await page.goto('/');
  await expect(page.getByRole('heading', {name: 'Agent Card Catalog'})).toBeVisible();
  await expect(page.getByText('API: configured', {exact: true})).toBeVisible();

  await createWorkspace(page);
  const shareA = await registerCard(page, runtimeA);
  const shareB = await registerCard(page, runtimeB);
  expect(shareA.publicUrl).toBe(`${publicAgentOrigin}/a/${shareA.publicAgentId}`);
  expect(shareB.publicUrl).toBe(`${publicAgentOrigin}/a/${shareB.publicAgentId}`);

  const releaseA = await publishTrustedRelease(page, runtimeA, leakTracker);
  const releaseB = await publishTrustedRelease(page, runtimeB, leakTracker);

  const ownerCatalogResponsePromise = page.waitForResponse((response) => {
    const url = new URL(response.url());
    return response.request().method() === 'GET' && url.pathname.endsWith('/v3/agents') && url.search === '';
  });
  await page.reload();
  const ownerCatalogResponse = await ownerCatalogResponsePromise;
  expect(ownerCatalogResponse.status()).toBe(200);
  const ownerCatalog = await ownerCatalogResponse.json() as {
    items: Array<{card: {agentId: string; version: string}; publicationStatus: string}>;
  };
  expect(ownerCatalog.items.map((item) => ({
    agentId: item.card.agentId,
    version: item.card.version,
    publicationStatus: item.publicationStatus,
  })), 'Owner discovery must expose both published runtime Cards').toEqual(expect.arrayContaining([
    {agentId: runtimeA.id, version: '1.0.0', publicationStatus: 'published'},
    {agentId: runtimeB.id, version: '1.0.0', publicationStatus: 'published'},
  ]));
  await expect(page.getByRole('heading', {name: 'Agent Card Catalog'})).toBeVisible();
  const publicResolutionRequests: string[] = [];
  const directPublicRequestPromise = page.waitForRequest((request) => request.method() === 'GET' && request.url().endsWith(`/v4/public/agents/${shareA.publicAgentId}`));
  await page.goto(`/a/${shareA.publicAgentId}`);
  const directPublicRequest = await directPublicRequestPromise;
  publicResolutionRequests.push(directPublicRequest.url());
  expect(directPublicRequest.headers().authorization).toBeUndefined();
  await expect(page.getByRole('heading', {name: 'Review a shared Agent'})).toBeVisible();
  const directPanel = page.locator('section').filter({hasText: 'Public Share'});
  const directReleaseSelect = directPanel.getByLabel('Exact public Release', {exact: true});
  await expect(directReleaseSelect).toHaveValue('');
  await directReleaseSelect.selectOption(releaseA.releaseId);
  const directInstallResponsePromise = page.waitForResponse((response) => response.url().includes(`/v3/workspaces/${workspaceId}/installations`) && response.request().method() === 'POST');
  await directPanel.getByRole('button', {name: 'Install exact Release', exact: true}).click();
  const directInstallResponse = await directInstallResponsePromise;
  expect(directInstallResponse.status()).toBe(201);
  const directInstallation = await directInstallResponse.json() as {installedReleaseId: string; agentId: string; versionConstraint: string};
  expect(directInstallation).toMatchObject({installedReleaseId: releaseA.releaseId, agentId: runtimeA.id, versionConstraint: '1.0.0'});
  await expect(directPanel.getByText(`Installed exact Release ${releaseA.releaseId}.`, {exact: true})).toBeVisible();

  await page.goto('/');
  await expect(page.getByRole('heading', {name: 'Agent Card Catalog'})).toBeVisible();
  await page.getByRole('button', {name: 'Installations', exact: true}).click();
  const publicPanel = page.locator('section').filter({hasText: 'Public Share'});
  await publicPanel.getByLabel('Public Agent URL', {exact: true}).fill(shareB.publicUrl);
  const pastedPublicRequestPromise = page.waitForRequest((request) => request.method() === 'GET' && request.url().endsWith(`/v4/public/agents/${shareB.publicAgentId}`));
  await publicPanel.getByRole('button', {name: 'Resolve', exact: true}).click();
  const pastedPublicRequest = await pastedPublicRequestPromise;
  publicResolutionRequests.push(pastedPublicRequest.url());
  expect(pastedPublicRequest.headers().authorization).toBeUndefined();
  const pastedReleaseSelect = publicPanel.getByLabel('Exact public Release', {exact: true});
  await expect(pastedReleaseSelect).toHaveValue('');
  await pastedReleaseSelect.selectOption(releaseB.releaseId);
  await expect(publicPanel.getByRole('checkbox', {name: /text\.read/})).not.toBeChecked();
  await publicPanel.getByRole('checkbox', {name: /text\.read/}).check();
  const pastedInstallResponsePromise = page.waitForResponse((response) => response.url().includes(`/v3/workspaces/${workspaceId}/installations`) && response.request().method() === 'POST');
  await publicPanel.getByRole('button', {name: 'Install exact Release', exact: true}).click();
  const pastedInstallResponse = await pastedInstallResponsePromise;
  expect(pastedInstallResponse.status()).toBe(201);
  const pastedInstallation = await pastedInstallResponse.json() as {installedReleaseId: string; agentId: string; acceptedPermissions: string[]};
  expect(pastedInstallation).toMatchObject({installedReleaseId: releaseB.releaseId, agentId: runtimeB.id, acceptedPermissions: ['text.read']});
  await expect(publicPanel.getByText(`Installed exact Release ${releaseB.releaseId}.`, {exact: true})).toBeVisible();
  expect(publicResolutionRequests).toEqual([`${apiBaseURL}/v4/public/agents/${shareA.publicAgentId}`, `${apiBaseURL}/v4/public/agents/${shareB.publicAgentId}`]);

  await page.getByRole('button', {name: 'Installations', exact: true}).click();
  await selectOptionContaining(page.getByLabel('Published Agent', {exact: true}), runtimeA.id);
  await page.getByLabel('Trusted Release ID', {exact: true}).fill('release-does-not-exist');
  const preflightResponsePromise = page.waitForResponse((response) => response.url().includes('/v4/releases/release-does-not-exist') && response.request().method() === 'GET');
  await page.getByRole('button', {name: 'Preflight', exact: true}).click();
  const preflightResponse = await preflightResponsePromise;
  expect(preflightResponse.status()).toBe(404);
  const preflightError = await preflightResponse.json() as {code: string; traceId: string};
  expect(preflightError.code).toBe('NOT_FOUND');
  expect(preflightError.traceId).toBeTruthy();
  const preflightHeaderTrace = preflightResponse.headers()['x-nek-trace-id'];
  if (preflightHeaderTrace !== undefined) expect(preflightHeaderTrace).toBe(preflightError.traceId);
  await expect(page.getByText(/NOT_FOUND/)).toBeVisible();
  await expect(page.getByText(/HTTP 404/)).toBeVisible();
  await expect(page.getByText(new RegExp('traceId: ' + escapeRegExp(preflightError.traceId)))).toBeVisible();

  await page.getByRole('button', {name: 'Invocations', exact: true}).click();
  const installationSelect = page.getByLabel('Installed Agent', {exact: true});
  await selectOptionContaining(installationSelect, runtimeB.id);
  await page.getByLabel('Capability', {exact: true}).fill(runtimeB.capability);
  await page.getByLabel('Input JSON', {exact: true}).fill(JSON.stringify({fixture: 'nested', value: {message: 'browser-json'}}));
  const jsonResponsePromise = page.waitForResponse((response) => response.url().includes('/v4/workspaces/' + workspaceId + '/invocations') && response.request().method() === 'POST' && (response.request().postData() ?? '').includes('"stream":false'));
  await page.getByRole('button', {name: 'Invoke', exact: true}).click();
  const jsonResponse = await jsonResponsePromise;
  const jsonResponseBody = await jsonResponse.text();
  if (jsonResponse.status() !== 200) await logInvocationTraceDiagnostic(page, jsonResponseBody);
  expect(jsonResponse.status(), `JSON invocation response: ${summarizeResponse(jsonResponse.status(), jsonResponseBody)}`).toBe(200);

  const response = page.locator('pre').filter({hasText: 'invocationId'}).last();
  await expect(response).toContainText('runtime-a');
  const result = JSON.parse((await response.textContent()) ?? '{}') as {invocationId: string; rootTaskId: string; traceId: string};
  expect(result.invocationId).toBeTruthy();
  expect(result.rootTaskId).toBeTruthy();
  expect(result.traceId).toBeTruthy();

  await page.getByRole('button', {name: 'Invocations', exact: true}).click();
  await selectOptionContaining(installationSelect, runtimeB.id);
  await page.getByLabel('Capability', {exact: true}).fill(runtimeB.capability);
  await page.getByLabel('Input JSON', {exact: true}).fill(JSON.stringify({fixture: 'stream-success', value: 'browser-sse'}));
  await page.getByLabel('Stream result over SSE', {exact: true}).check();
  const sseResponsePromise = page.waitForResponse((response) => response.url().includes('/v4/workspaces/' + workspaceId + '/invocations') && response.request().method() === 'POST' && (response.request().postData() ?? '').includes('"stream":true'));
  await page.getByRole('button', {name: 'Invoke', exact: true}).click();
  const sseResponse = await sseResponsePromise;
  expect(sseResponse.status()).toBe(200);
  assertResultStream(await sseResponse.text());
  await expect(page.getByText('#0 accepted', {exact: true})).toBeVisible();
  await expect(page.getByText(/completed/, {exact: true}).last()).toBeVisible();

  await page.getByRole('button', {name: 'Ledger', exact: true}).click();
  await page.getByLabel('Trace ID', {exact: true}).fill(result.traceId);
  const traceResponsePromise = page.waitForResponse((response) => response.url().includes('/v4/workspaces/' + workspaceId + '/traces/' + result.traceId) && response.request().method() === 'GET');
  await page.getByRole('button', {name: 'Read', exact: true}).last().click();
  const traceResponse = await traceResponsePromise;
  expect(traceResponse.status()).toBe(200);
  const tracePayload = await traceResponse.json() as {
    traceId: string;
    invocations: Array<{invocationId: string; parentInvocationId?: string; rootTaskId: string; traceId: string; targetAgentId: string}>;
  };
  const rootInvocation = tracePayload.invocations.find((invocation) => invocation.invocationId === result.invocationId);
  const childInvocation = tracePayload.invocations.find((invocation) => invocation.targetAgentId === runtimeA.id && invocation.invocationId !== result.invocationId);
  expect(tracePayload.traceId).toBe(result.traceId);
  expect(rootInvocation).toBeDefined();
  expect(childInvocation).toBeDefined();
  expect(childInvocation?.parentInvocationId).toBe(rootInvocation?.invocationId);
  expect(childInvocation?.rootTaskId).toBe(rootInvocation?.rootTaskId);
  expect(childInvocation?.traceId).toBe(rootInvocation?.traceId);
  await expect(page.getByText(new RegExp(`${escapeRegExp(result.traceId)}`)).last()).toBeVisible();
  const ledgerText = await page.locator('main').innerText();
  expect(ledgerText).toContain(runtimeA.id);
  expect(ledgerText).toContain(runtimeB.id);
  expect(ledgerText).toContain(result.invocationId);
  expect(ledgerText).toContain(releaseA.releaseId);
  expect(ledgerText).toContain(releaseB.releaseId);
  expect(ledgerText).toContain(releaseA.cardDigest);
  expect(ledgerText).toContain(releaseB.cardDigest);

  const gatewayOrigin = new URL(apiBaseURL).origin;
  expect(apiRequests.length).toBeGreaterThan(0);
  expect(apiRequests.every((url) => new URL(url).origin === gatewayOrigin)).toBe(true);
  expect(requestUrls.some((url) => {
    const parsed = new URL(url);
    return /\/internal\/|\/agent\//.test(parsed.pathname) || parsed.hostname === 'runtime-a' || parsed.hostname === 'runtime-b';
  })).toBe(false);

  apiRequests.length = 0;
  for (const {hash, marker} of [
    {hash: '#/demo', marker: 'Three directions. Same data. Pick one.'},
    {hash: '#/demo/glass', marker: '6 cards'},
    {hash: '#/demo/terminal', marker: 'NEKIRO//OPS'},
    {hash: '#/demo/saas', marker: 'Find the right Agent for every workflow'},
  ]) {
    await page.goto('/' + hash);
    await expect(page.getByText(marker, {exact: true})).toBeVisible();
  }
  expect(apiRequests).toEqual([]);
});

async function createWorkspace(page: Page): Promise<void> {
  const input = page.locator('input[placeholder="workspace id"]');
  await input.fill(workspaceId);
  await page.getByRole('button', {name: 'Create workspace'}).click();
  await expect(page.getByText(`Workspace: ${workspaceId}`, {exact: true})).toBeVisible();
}

async function registerCard(page: Page, fixture: AgentFixture): Promise<PublicShare> {
  await page.getByRole('button', {name: 'Registry', exact: true}).click();
  await page.getByRole('button', {name: 'Register Agent Card', exact: true}).click();
  await page.getByLabel('Agent ID', {exact: true}).fill(fixture.id);
  await page.getByLabel('Name', {exact: true}).fill(fixture.name);
  await page.getByLabel('Owner ID', {exact: true}).fill(providerId);
  await page.getByLabel('Owner display name', {exact: true}).fill('Browser Provider');
  await page.getByLabel('Version', {exact: true}).fill('1.0.0');
  await page.getByLabel('A2A endpoint', {exact: true}).fill(fixture.endpoint);
  await page.getByLabel('Authentication', {exact: true}).selectOption('http_bearer');
  await page.getByLabel('Permissions, one per line as ID: description', {exact: true}).fill((fixture.permissions ?? []).map((permission) => `${permission}: ${permission}`).join('\n'));
  await page.getByLabel('Capabilities JSON', {exact: true}).fill(JSON.stringify({capabilities: [
    {id: fixture.capability, name: fixture.capability, description: 'Browser acceptance capability', inputSchema: {type: 'object'}, outputSchema: {type: 'object'}, requiredPermissions: fixture.permissions ?? []},
  ]}, null, 2));
  const responsePromise = page.waitForResponse((response) => response.url().endsWith('/v3/agents') && response.request().method() === 'POST');
  await page.getByRole('button', {name: 'Submit draft', exact: true}).click();
  const response = await responsePromise;
  expect(response.status()).toBe(201);
  const body = await response.json() as {publicAgentId?: string; publicUrl?: string};
  expect(body.publicAgentId).toMatch(/^agt_[0-9a-f]{32}$/);
  expect(body.publicUrl).toBe(`${publicAgentOrigin}/a/${body.publicAgentId}`);
  await expect(page.getByText(fixture.id, {exact: true}).first()).toBeVisible();
  await expect(page.getByRole('link', {name: body.publicUrl, exact: true})).toBeVisible();
  return {publicAgentId: body.publicAgentId as string, publicUrl: body.publicUrl as string};
}

async function publishTrustedRelease(page: Page, fixture: AgentFixture, leakTracker: BrowserLeakTracker): Promise<ReleaseEvidence> {
  await page.getByRole('button', {name: 'Trusted Publication', exact: true}).click();
  await page.getByRole('button', {name: new RegExp(escapeRegExp(fixture.id))}).first().click();
  await page.getByLabel('Agent endpoint', {exact: true}).fill(fixture.endpoint);
  await page.getByRole('button', {name: 'Create Binding', exact: true}).click();
  await expect(page.getByText('pending', {exact: true}).last()).toBeVisible();

  await page.getByRole('button', {name: 'Issue Challenge', exact: true}).click();
  const challengeId = await textMatching(page, /^challenge-[A-Za-z0-9._:-]+$/);
  const proof = (await page.locator('code').last().textContent())?.trim();
  if (!proof) throw new Error('Console did not render the one-time challenge proof');
  const persistedValues = await page.evaluate(() => [
    ...Object.entries(localStorage),
    ...Object.entries(sessionStorage),
  ].flat());
  expect(persistedValues).not.toContain(proof);
  injectChallengeProof(fixture.service, challengeId, proof);
  await page.getByRole('button', {name: 'Complete Verification', exact: true}).click();
  await expect(page.getByText('verified', {exact: true}).last()).toBeVisible();
  await expect(page.locator('code')).toHaveCount(0);
  expect(leakTracker.requestUrls.some((url) => url.includes(proof))).toBe(false);
  expect(leakTracker.requestBodies.some((body) => body.includes(proof))).toBe(false);
  expect(leakTracker.consoleMessages.some((message) => message.includes(proof))).toBe(false);

  await page.getByRole('button', {name: 'Create Release', exact: true}).click();
  const releaseSection = page.locator('section').filter({hasText: '3. Immutable Release'});
  const releaseState = releaseSection.getByText(/^(pending_verification|verified)$/, {exact: true}).last();
  await expect(releaseState).toBeVisible();
  if ((await releaseState.textContent()) === 'pending_verification') {
    const verifyButton = releaseSection.getByRole('button', {name: 'Verify', exact: true});
    await expect(verifyButton).toBeEnabled();
    await verifyButton.click();
  }
  await expect(releaseSection.getByText('verified', {exact: true}).last()).toBeVisible();
  await releaseSection.getByRole('button', {name: 'Publish', exact: true}).click();
  await expect(releaseSection.getByText('published', {exact: true}).last()).toBeVisible();

  const releaseId = await readFactValue(releaseSection, 'Release');
  const cardDigest = await readFactValue(releaseSection, 'Card digest');
  if (!/^[A-Za-z0-9._:-]+$/.test(releaseId) || !/^[0-9a-f]{64}$/.test(cardDigest)) {
    throw new Error('Console did not render immutable Release provenance');
  }
  return {releaseId, cardDigest};
}

async function installRelease(page: Page, fixture: AgentFixture, releaseId: string): Promise<void> {
  await page.getByRole('button', {name: 'Installations', exact: true}).click();
  const agentSelect = page.getByLabel('Published Agent', {exact: true});
  await selectOptionContaining(agentSelect, fixture.id);
  await page.getByLabel('Trusted Release ID', {exact: true}).fill(releaseId);
  await page.getByRole('button', {name: 'Preflight', exact: true}).click();
  await expect(page.getByText('Published Release preflight passed', {exact: true})).toBeVisible();
  await page.getByRole('button', {name: 'Install exact pin', exact: true}).click();
  await expect(page.getByText(releaseId, {exact: true}).last()).toBeVisible();
}

async function selectOptionContaining(select: Locator, text: string): Promise<void> {
  await expect.poll(
    async () => select.locator('option').evaluateAll((options, wanted) => options.some((item) => {
      const option = item as HTMLOptionElement;
      return option.textContent?.includes(String(wanted)) || option.value.includes(String(wanted));
    }), text),
    {message: `Expected an Agent option containing ${text}`},
  ).toBe(true);
  const value = await select.locator('option').evaluateAll((options, wanted) => {
    const option = options.find((item) => {
      const candidate = item as HTMLOptionElement;
      return candidate.textContent?.includes(String(wanted)) || candidate.value.includes(String(wanted));
    });
    if (!option) throw new Error(`No select option contains ${String(wanted)}`);
    return (option as HTMLOptionElement).value;
  }, text);
  await select.selectOption(value);
}

async function textMatching(page: Page, pattern: RegExp): Promise<string> {
  const value = (await page.getByText(pattern).last().textContent())?.trim();
  if (!value) throw new Error(`Console did not render text matching ${pattern}`);
  return value;
}

async function readFactValue(section: Locator, label: string): Promise<string> {
  const labelElement = section.getByText(label, {exact: true});
  return ((await labelElement.locator('..').locator('div').nth(1).textContent()) ?? '').trim();
}

function injectChallengeProof(service: string, challengeId: string, proof: string): void {
  execFileSync('docker', [
    'compose', '--project-name', composeProject, '--file', composeFile,
    'exec', '-T', service, 'sh', '-c',
    'umask 077; cat > "$NEKIRO_AGENT_CHALLENGE_DIRECTORY/$1"', 'sh', challengeId,
  ], {input: proof, encoding: 'utf8', stdio: ['pipe', 'ignore', 'pipe']});
}

function assertResultStream(body: string): void {
  const events = body.trim().split(/\r?\n\r?\n/).filter(Boolean).map((block) => {
    const line = block.split(/\r?\n/).find((value) => value.startsWith('data: '));
    if (!line) throw new Error('SSE response omitted a data line');
    return JSON.parse(line.slice('data: '.length)) as {
      schemaVersion: string;
      sequence: number;
      type: string;
      status: string;
      invocationId: string;
      rootTaskId: string;
      traceId: string;
    };
  });
  if (events.length < 2) throw new Error('SSE response did not contain accepted and terminal events');
  const first = events[0];
  const last = events[events.length - 1];
  if (first.type !== 'accepted' || first.status !== 'pending' || first.sequence !== 0) throw new Error('SSE response did not begin with accepted/pending sequence 0');
  events.forEach((event, index) => {
    if (event.schemaVersion !== '2' || event.sequence !== index || event.invocationId !== first.invocationId || event.rootTaskId !== first.rootTaskId || event.traceId !== first.traceId) {
      throw new Error('SSE response correlation or sequence changed');
    }
  });
  if (last.type !== 'completed' || last.status !== 'succeeded') throw new Error('SSE response did not end with completed/succeeded');
}

function summarizeResponse(status: number, body: string): string {
  let value: unknown;
  try {
    value = JSON.parse(body);
  } catch {
    return `status=${status}, body=non-json`;
  }
  if (!value || typeof value !== 'object' || Array.isArray(value)) {
    return `status=${status}, body=non-object-json`;
  }
  const record = value as Record<string, unknown>;
  const keys = Object.keys(record).sort().join(',');
  const safeFields = ['code', 'traceId', 'invocationId', 'rootTaskId']
    .filter((key) => typeof record[key] === 'string')
    .map((key) => `${key}=${record[key] as string}`)
    .join(',');
  return `status=${status}, keys=${keys}${safeFields ? ', ' + safeFields : ''}`;
}

async function logInvocationTraceDiagnostic(page: Page, body: string): Promise<void> {
  let value: unknown;
  try {
    value = JSON.parse(body);
  } catch {
    console.log('JSON invocation trace diagnostic: error_body=non-json');
    return;
  }
  if (!value || typeof value !== 'object' || Array.isArray(value)) {
    console.log('JSON invocation trace diagnostic: error_body=non-object-json');
    return;
  }
  const error = value as Record<string, unknown>;
  const traceId = typeof error.traceId === 'string' ? error.traceId : '';
  if (!traceId) {
    console.log('JSON invocation trace diagnostic: trace_id=missing');
    return;
  }
  try {
    const response = await page.request.get(`${apiBaseURL}/v4/workspaces/${encodeURIComponent(workspaceId)}/traces/${encodeURIComponent(traceId)}`, {
      headers: {Authorization: `Bearer ${ownerToken}`, Accept: 'application/json'},
    });
    const traceBody = await response.text();
    console.log(`JSON invocation trace diagnostic: ${summarizeTrace(response.status(), traceBody)}`);
  } catch (diagnosticError) {
    const errorName = diagnosticError instanceof Error ? diagnosticError.name : 'unknown';
    console.log(`JSON invocation trace diagnostic: request_error=${errorName}`);
  }
}

function summarizeTrace(status: number, body: string): string {
  let value: unknown;
  try {
    value = JSON.parse(body);
  } catch {
    return `status=${status}, body=non-json`;
  }
  if (!value || typeof value !== 'object' || Array.isArray(value)) {
    return `status=${status}, body=non-object-json`;
  }
  const record = value as Record<string, unknown>;
  const invocations = Array.isArray(record.invocations) ? record.invocations : [];
  const states = invocations.filter((item): item is Record<string, unknown> => Boolean(item && typeof item === 'object' && !Array.isArray(item)))
    .map((item) => [
      diagnosticIdentifier(item.invocationId),
      item.parentInvocationId === undefined ? '' : diagnosticIdentifier(item.parentInvocationId),
      diagnosticIdentifier(item.targetAgentId),
      diagnosticEnum(item.status, ['pending', 'routing', 'running', 'succeeded', 'failed', 'canceled', 'timed_out']),
      item.errorCode === undefined ? '' : diagnosticEnum(item.errorCode, ['VALIDATION_ERROR', 'UNAUTHENTICATED', 'FORBIDDEN', 'NOT_FOUND', 'CONFLICT', 'NOT_ACCEPTABLE', 'PAYLOAD_TOO_LARGE', 'AGENT_NOT_INSTALLED', 'INSTALLATION_DISABLED', 'AGENT_DISABLED', 'AGENT_RELEASE_UNPUBLISHED', 'AGENT_RELEASE_SUSPENDED', 'AGENT_RELEASE_REVOKED', 'CAPABILITY_NOT_ALLOWED', 'ROUTE_NOT_FOUND', 'AGENT_AUTH_UNSUPPORTED', 'AGENT_RESPONSE_TOO_LARGE', 'A2A_PROTOCOL_ERROR', 'AGENT_UNAVAILABLE', 'AGENT_EXECUTION_FAILED', 'DEPENDENCY_ERROR', 'TIMEOUT', 'CANCELED', 'INTERNAL_ERROR']),
    ].join('/'))
    .join(';');
  return `status=${status}, invocation_states=${states || 'none'}`;
}

function diagnosticIdentifier(value: unknown): string {
  return typeof value === 'string' && /^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$/.test(value) ? value : 'invalid';
}

function diagnosticEnum(value: unknown, allowed: string[]): string {
  return typeof value === 'string' && allowed.includes(value) ? value : 'invalid';
}

function required(name: string): string {
  const value = process.env[name];
  if (!value || value !== value.trim()) throw new Error(`${name} is required and must not contain surrounding whitespace`);
  return value;
}

function escapeRegExp(value: string): string {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}
