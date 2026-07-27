import {execFileSync} from 'node:child_process';

import {expect, test, type Locator, type Page} from '@playwright/test';

const providerId = required('VITE_NEKIRO_PROVIDER_ID');
const workspaceId = required('VITE_NEKIRO_DEFAULT_WORKSPACE_ID');
const composeFile = required('NEKIRO_E2E_COMPOSE_FILE');
const composeProject = required('NEKIRO_E2E_COMPOSE_PROJECT');

type AgentFixture = {
  id: string;
  name: string;
  endpoint: string;
  service: string;
  capability: string;
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
};

const runtimeB: AgentFixture = {
  id: 'runtime-b',
  name: 'Browser Runtime B',
  endpoint: 'http://runtime-b:8092',
  service: 'runtime-b',
  capability: 'runtime.cross',
};

test.describe.configure({mode: 'serial'});

test('production Console completes trusted publication, invocation, trace, and isolated demos', async ({page}) => {
  const apiRequests: string[] = [];
  const requestUrls: string[] = [];
  const requestBodies: string[] = [];
  const consoleMessages: string[] = [];
  const leakTracker: BrowserLeakTracker = {requestUrls, requestBodies, consoleMessages};
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
  await registerCard(page, runtimeA);
  await registerCard(page, runtimeB);

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
  await installRelease(page, runtimeA, releaseA.releaseId);
  await installRelease(page, runtimeB, releaseB.releaseId);

  await page.getByRole('button', {name: 'Installations', exact: true}).click();
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
  await page.getByRole('button', {name: 'Invoke', exact: true}).click();

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
  await page.getByRole('button', {name: 'Read', exact: true}).last().click();
  await expect(page.getByText(new RegExp(`${escapeRegExp(result.traceId)}`)).last()).toBeVisible();
  const ledgerText = await page.locator('main').innerText();
  expect(ledgerText).toContain(runtimeA.id);
  expect(ledgerText).toContain(runtimeB.id);
  expect(ledgerText).toContain(result.invocationId);
  expect(ledgerText).toContain(releaseA.releaseId);
  expect(ledgerText).toContain(releaseB.releaseId);
  expect(ledgerText).toContain(releaseA.cardDigest);
  expect(ledgerText).toContain(releaseB.cardDigest);

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

async function registerCard(page: Page, fixture: AgentFixture): Promise<void> {
  await page.getByRole('button', {name: 'Registry', exact: true}).click();
  await page.getByRole('button', {name: 'Register Agent Card', exact: true}).click();
  await page.getByLabel('Agent ID', {exact: true}).fill(fixture.id);
  await page.getByLabel('Name', {exact: true}).fill(fixture.name);
  await page.getByLabel('Owner ID', {exact: true}).fill(providerId);
  await page.getByLabel('Owner display name', {exact: true}).fill('Browser Provider');
  await page.getByLabel('Version', {exact: true}).fill('1.0.0');
  await page.getByLabel('A2A endpoint', {exact: true}).fill(fixture.endpoint);
  await page.getByLabel('Authentication', {exact: true}).selectOption('http_bearer');
  await page.getByLabel('Capabilities JSON', {exact: true}).fill(JSON.stringify({capabilities: [
    {id: fixture.capability, name: fixture.capability, description: 'Browser acceptance capability', inputSchema: {type: 'object'}, outputSchema: {type: 'object'}, requiredPermissions: []},
  ]}, null, 2));
  await page.getByRole('button', {name: 'Submit draft', exact: true}).click();
  await expect(page.getByText(fixture.id, {exact: true}).first()).toBeVisible();
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

function required(name: string): string {
  const value = process.env[name];
  if (!value || value !== value.trim()) throw new Error(`${name} is required and must not contain surrounding whitespace`);
  return value;
}

function escapeRegExp(value: string): string {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}
