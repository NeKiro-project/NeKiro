import assert from 'node:assert/strict';
import {test} from 'node:test';

import {
  buildAgentCard,
  mapCatalogEntry,
  NekiroApiClient,
  NekiroApiError,
  validateTrustedInstallation,
  type AgentCardV02,
  type AgentRelease,
  type CatalogEntry,
} from './nekiro';

const limits = {
  timeoutMs: 30_000,
  maxInputBytes: 1_048_576,
  maxOutputBytes: 1_048_576,
  streaming: true,
};

test('buildAgentCard converts registration fields into an Agent Card v0.2 payload', () => {
  const card = buildAgentCard({
    agentId: 'runtime.echo',
    name: 'Runtime Echo Agent',
    ownerId: 'team.platform',
    ownerDisplayName: 'Platform Team',
    description: 'Echoes structured input.',
    version: '1.0.0',
    endpoint: 'http://127.0.0.1:9000/a2a',
    authentication: 'none',
    permissions: [{id: 'READ_LOGS', description: 'Read logs.'}],
    capabilitiesJson: JSON.stringify({
      capabilities: [{
        id: 'runtime.echo',
        name: 'Runtime Echo',
        description: 'Echo input.',
        inputSchema: {type: 'object'},
        outputSchema: {type: 'object'},
        requiredPermissions: ['READ_LOGS'],
      }],
    }),
    limits,
  });

  assert.deepEqual(card, {
    schemaVersion: '0.2',
    agentId: 'runtime.echo',
    name: 'Runtime Echo Agent',
    description: 'Echoes structured input.',
    owner: {id: 'team.platform', displayName: 'Platform Team'},
    version: '1.0.0',
    protocol: {type: 'a2a', version: '0.3.0', transport: 'JSONRPC', endpoint: 'http://127.0.0.1:9000/a2a'},
    skills: [{
      id: 'runtime.echo',
      name: 'Runtime Echo',
      description: 'Echo input.',
      inputSchema: {type: 'object'},
      outputSchema: {type: 'object'},
      requiredPermissions: ['READ_LOGS'],
    }],
    authentication: {type: 'none'},
    permissions: [{id: 'READ_LOGS', description: 'Read logs.'}],
    limits,
  });
});

test('buildAgentCard rejects duplicate capabilities and undeclared required permissions', () => {
  assert.throws(
    () => buildAgentCard({
      agentId: 'runtime.echo',
      name: 'Runtime Echo',
      ownerId: 'team.platform',
      ownerDisplayName: 'Platform Team',
      description: 'Echoes structured input.',
      version: '1.0.0',
      endpoint: 'http://127.0.0.1:9000/a2a',
      authentication: 'none',
      permissions: [],
      capabilitiesJson: JSON.stringify({capabilities: [{id: 'same', name: 'Same', description: 'Same.', inputSchema: {}, outputSchema: {}, requiredPermissions: []}, {id: 'same', name: 'Same', description: 'Same.', inputSchema: {}, outputSchema: {}, requiredPermissions: []}]}),
      limits,
    }),
    /duplicate capability id/i,
  );

  assert.throws(
    () => buildAgentCard({
      agentId: 'runtime.echo',
      name: 'Runtime Echo',
      ownerId: 'team.platform',
      ownerDisplayName: 'Platform Team',
      description: 'Echoes structured input.',
      version: '1.0.0',
      endpoint: 'http://127.0.0.1:9000/a2a',
      authentication: 'none',
      permissions: [],
      capabilitiesJson: JSON.stringify({capabilities: [{id: 'runtime.echo', name: 'Echo', description: 'Echo.', inputSchema: {}, outputSchema: {}, requiredPermissions: ['READ_LOGS']}]}),
      limits,
    }),
    /not declared/i,
  );
});

test('mapCatalogEntry maps Catalog entries to the Console view model without deprecated state', () => {
  const card: AgentCardV02 = {
    schemaVersion: '0.2',
    agentId: 'runtime.disabled',
    name: 'Disabled Runtime',
    description: 'A disabled runtime.',
    owner: {id: 'team.platform', displayName: 'Platform Team'},
    version: '2.0.0',
    protocol: {type: 'a2a', version: '0.3.0', transport: 'JSONRPC', endpoint: 'http://127.0.0.1:9000/a2a'},
    skills: [
      {id: 'runtime.echo', name: 'Echo', description: 'Echoes input.', inputSchema: {type: 'object'}, outputSchema: {type: 'object'}, requiredPermissions: []},
      {id: 'runtime.inspect', name: 'Inspect', description: 'Inspects input.', inputSchema: {type: 'object'}, outputSchema: {type: 'object'}, requiredPermissions: []},
    ],
    authentication: {type: 'none'},
    permissions: [{id: 'READ_LOGS', description: 'Read logs.'}],
    limits,
  };
  const entry: CatalogEntry = {
    card,
    publicationStatus: 'disabled',
    registeredAt: '2026-07-14T00:00:00Z',
  };

  const agent = mapCatalogEntry(entry);

  assert.equal(agent.id, 'runtime.disabled');
  assert.equal(agent.owner, 'Platform Team');
  assert.equal(agent.ownerId, 'team.platform');
  assert.equal(agent.version, '2.0.0');
  assert.equal(agent.status, 'disabled');
  assert.deepEqual(agent.tags, ['runtime.echo', 'runtime.inspect']);
  assert.deepEqual(agent.permissions, [{id: 'READ_LOGS', description: 'Read logs.'}]);
  assert.equal(JSON.parse(agent.schema).agentId, 'runtime.disabled');
});

test('NekiroApiClient sends v3 Catalog search requests with auth and decodes platform errors', async () => {
  const requests: Array<{url: string; init?: RequestInit}> = [];
  const client = new NekiroApiClient({
    baseUrl: 'https://api.example.test/',
    token: 'test-token',
    fetchImpl: async (input, init) => {
      requests.push({url: String(input), init});
      return new Response(JSON.stringify({
        code: 'CONFLICT',
        message: 'The requested operation conflicts with current state.',
        traceId: 'trace-1',
      }), {
        status: 409,
        headers: {'Content-Type': 'application/json'},
      });
    },
  });

  await assert.rejects(
    () => client.searchAgents({query: 'echo'}),
    (error: unknown) => {
      assert.ok(error instanceof NekiroApiError);
      assert.equal(error.status, 409);
      assert.equal(error.code, 'CONFLICT');
      assert.equal(error.traceId, 'trace-1');
      return true;
    },
  );

  assert.equal(requests[0]?.url, 'https://api.example.test/v3/agents?query=echo');
  const headers = new Headers(requests[0]?.init?.headers);
  assert.equal(headers.get('Accept'), 'application/json');
  assert.equal(headers.get('Authorization'), 'Bearer test-token');
});

test('NekiroApiClient resolves public Agent shares anonymously and preserves exact Release facts', async () => {
  const requests: Array<{url: string; init?: RequestInit}> = [];
  const publicAgentId = 'agt_0123456789abcdef0123456789abcdef';
  const release = {
    releaseId: 'release-1', agentId: 'agent.echo', name: 'Echo', description: 'Public echo',
    owner: {id: 'owner-a', displayName: 'Owner A'}, agentCardVersion: '1.2.3',
    cardDigest: '0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef',
    publishedAt: '2026-07-30T00:01:00Z', authenticationType: 'none',
    skills: [{id: 'runtime.echo', name: 'Echo', description: 'Echo input.', inputSchema: {type: 'object'}, outputSchema: {type: 'object'}, requiredPermissions: []}],
    permissions: [], limits,
  };
  const client = new NekiroApiClient({
    baseUrl: 'https://api.example.test', anonymousOnly: true, publicAgentOrigin: 'https://agents.example.test',
    fetchImpl: async (input, init) => {
      requests.push({url: String(input), init});
      return new Response(JSON.stringify({schemaVersion: '1', publicAgentId, publicUrl: 'https://agents.example.test/a/' + publicAgentId, registeredAt: '2026-07-30T00:00:00Z', availability: 'installable', releases: [release]}), {status: 200, headers: {'Content-Type': 'application/json'}});
    },
  });
  const result = await client.resolvePublicAgent(publicAgentId);
  assert.equal(result.releases[0]?.releaseId, 'release-1');
  assert.equal(requests[0]?.url, 'https://api.example.test/v4/public/agents/' + publicAgentId);
  const headers = new Headers(requests[0]?.init?.headers);
  assert.equal(headers.get('Authorization'), null);
});

test('NekiroApiClient covers Workspace and Installation v3 paths', async () => {
  const requests: Array<{url: string; init?: RequestInit}> = [];
  const client = new NekiroApiClient({
    baseUrl: 'https://api.example.test',
    token: 'test-token',
    fetchImpl: async (input, init) => {
      requests.push({url: String(input), init});
      return new Response(JSON.stringify({items: [{
        installationId: 'installation-1', workspaceId: 'workspace.alpha', agentId: 'agent.echo', versionConstraint: '1.2.3', installedVersion: '1.2.3', installedReleaseId: 'release-1', acceptedPermissions: [], status: 'enabled', installedAt: '2026-07-26T00:00:00Z', updatedAt: '2026-07-26T00:00:00Z',
      }]}), {status: 200, headers: {'Content-Type': 'application/json'}});
    },
  });

  const result = await client.listInstallations('workspace.alpha', {limit: 50, cursor: 'next'});

  assert.equal(result.items[0]?.installedReleaseId, 'release-1');
  assert.equal(requests[0]?.url, 'https://api.example.test/v3/workspaces/workspace.alpha/installations?limit=50&cursor=next');
});

test('NekiroApiClient strictly maps every Installation read response', async () => {
  const installation = {
    installationId: 'installation-1', workspaceId: 'workspace.alpha', agentId: 'agent.echo', versionConstraint: '1.2.3', installedVersion: '1.2.3', installedReleaseId: 'release-1', acceptedPermissions: [], status: 'enabled', installedAt: '2026-07-26T00:00:00Z', updatedAt: '2026-07-26T00:00:00Z',
  };
  let response: Record<string, unknown> = installation;
  const client = new NekiroApiClient({
    baseUrl: 'https://api.example.test', token: 'owner-token',
    fetchImpl: async () => new Response(JSON.stringify(response), {status: 200, headers: {'Content-Type': 'application/json'}}),
  });
  assert.equal((await client.getInstallation('workspace.alpha', 'installation-1')).installationId, 'installation-1');
  assert.equal((await client.updateInstallation('workspace.alpha', 'installation-1', 'disabled')).status, 'enabled');
  assert.equal((await client.uninstallAgent('workspace.alpha', 'installation-1')).installedReleaseId, 'release-1');
  response = {...installation, unexpected: true};
  await assert.rejects(() => client.getInstallation('workspace.alpha', 'installation-1'), /unknown field/);
});

test('NekiroApiClient enforces Installation v2 semantic response rules', async () => {
  const base = {
    installationId: 'installation-1', workspaceId: 'workspace.alpha', agentId: 'agent.echo', versionConstraint: '^1.0.0', installedVersion: '1.2.3', acceptedPermissions: ['read', 'write'], status: 'enabled', installedAt: '2026-07-26T00:00:00Z', updatedAt: '2026-07-26T00:00:00Z',
  };
  let response: Record<string, unknown> = base;
  const client = new NekiroApiClient({baseUrl: 'https://api.example.test', token: 'owner-token', fetchImpl: async () => new Response(JSON.stringify(response), {status: 200, headers: {'Content-Type': 'application/json'}})});
  assert.equal((await client.getInstallation('workspace.alpha', 'installation-1')).installedVersion, '1.2.3');
  response = {...base, installedVersion: '2.0.0'};
  await assert.rejects(() => client.getInstallation('workspace.alpha', 'installation-1'), /does not satisfy/);
  response = {...base, acceptedPermissions: ['READ LOGS']};
  await assert.rejects(() => client.getInstallation('workspace.alpha', 'installation-1'), /safe identifier/);
  response = {...base, status: 'uninstalled', uninstalledAt: '2026-07-26T00:01:00Z'};
  await assert.rejects(() => client.getInstallation('workspace.alpha', 'installation-1'), /uninstalledAt must equal/);
  response = {...base, installedAt: '2026-07-26T00:01:00Z', updatedAt: '2026-07-25T00:00:00Z'};
  await assert.rejects(() => client.getInstallation('workspace.alpha', 'installation-1'), /must not precede/);
});

test('NekiroApiClient evaluates the active SemVer range forms and prerelease rule', async () => {
  const base = {
    installationId: 'installation-1', workspaceId: 'workspace.alpha', agentId: 'agent.echo', versionConstraint: '1.2.3', installedVersion: '1.2.3', acceptedPermissions: [], status: 'enabled', installedAt: '2026-07-26T00:00:00Z', updatedAt: '2026-07-26T00:00:00Z',
  };
  let response: Record<string, unknown> = base;
  const client = new NekiroApiClient({baseUrl: 'https://api.example.test', token: 'owner-token', fetchImpl: async () => new Response(JSON.stringify(response), {status: 200, headers: {'Content-Type': 'application/json'}})});
  for (const [versionConstraint, installedVersion, expected] of [
    ['1.2.3 - 2.3', '2.3.9', true],
    ['~1.2.3', '1.2.9', true],
    ['^0.2.3', '0.3.0', false],
    ['>= 1.2.3 < 2.0.0', '1.5.0', true],
    ['>= 1.2.3 < 2.0.0', '1.2.3-alpha', false],
    ['>= 1.2.3-alpha < 2.0.0', '1.2.3-beta', true],
    ['1.2.x || 2.0.0', '2.0.0', true],
    ['>=0-0', '0.0.0-alpha', true],
    ['>=0.0-0', '0.0.0-alpha', true],
    ['~1.1-alpha', '1.1.5', true],
    ['>=1.2.3-alpha < 2.0.0', '1.3.0-alpha', false],
    ['!=1.x', '1.2.3', false],
    ['!=1.x', '2.0.0', true],
    ['!=1.2.x', '1.2.9', false],
    ['!=1.2.x', '1.3.0', true],
    ['1.2.3 ||', '1.2.3', false],
    ['1.2.3,,2.0.0', '1.2.3', false],
    ['01.2.3', '1.2.3', false],
    ['999999999999999999999999.1.1', '1.2.3', false],
    ['9007199254740992.0.0', '9007199254740992.0.0', true],
    ['>=9007199254740992.0.0 <9007199254740994.0.0', '9007199254740993.0.0', true],
    ['>=9007199254740992.0.0 <9007199254740994.0.0', '9007199254740994.0.0', false],
    ['^9007199254740992.0.0', '9007199254740992.1.0', true],
    ['^9007199254740992.0.0', '9007199254740993.0.0', false],
    ['9007199254740991.0.0', '9007199254740991.0.0', true],
    ['^9007199254740991.0.0', '9007199254740991.0.1', true],
    ['^9007199254740991.0.0', '9007199254740992.0.0', false],
    ['9007199254740991.x', '9007199254740991.1.0', true],
    ['9007199254740991.x', '9007199254740992.0.0', false],
    ['18446744073709551615.0.0', '18446744073709551615.0.0', true],
    ['18446744073709551616.0.0', '18446744073709551616.0.0', false],
  ] as const) {
    response = {...base, versionConstraint, installedVersion};
    if (expected) {
      await assert.doesNotReject(() => client.getInstallation('workspace.alpha', 'installation-1'));
    } else {
      await assert.rejects(() => client.getInstallation('workspace.alpha', 'installation-1'), /does not satisfy/);
    }
  }
});

test('NekiroApiClient enforces Agent Card semantic rules on Catalog responses', async () => {
  let card = catalogCard();
  const client = new NekiroApiClient({
    baseUrl: 'https://api.example.test',
    token: 'owner-token',
    fetchImpl: async () => new Response(JSON.stringify({items: [{card, publicationStatus: 'published', registeredAt: '2026-07-26T00:00:00Z'}]}), {status: 200, headers: {'Content-Type': 'application/json'}}),
  });
  assert.equal((await client.searchAgents()).items.length, 1);

  card = {...catalogCard(), skills: []};
  await assert.rejects(() => client.searchAgents(), /skills must contain at least one/);
  card = {...catalogCard(), skills: [catalogCard().skills[0], {...catalogCard().skills[0], id: 'runtime.echo'}]};
  await assert.rejects(() => client.searchAgents(), /duplicate skill id/);
  card = {...catalogCard(), permissions: [{id: 'READ_LOGS', description: 'Read logs.'}, {id: 'READ_LOGS', description: 'Duplicate.'}]};
  await assert.rejects(() => client.searchAgents(), /duplicate permission id/);
  card = {...catalogCard(), skills: [{...catalogCard().skills[0], requiredPermissions: ['MISSING']}]};
  await assert.rejects(() => client.searchAgents(), /not declared/);
  card = {...catalogCard(), name: 'x'.repeat(121)};
  await assert.rejects(() => client.searchAgents(), /non-empty string/);
});

test('NekiroApiClient installs an exact trusted version and preserves Release provenance', async () => {
  const requests: Array<{url: string; init?: RequestInit}> = [];
  const installation = {
    installationId: 'installation-1',
    workspaceId: 'workspace.alpha',
    agentId: 'agent.echo',
    versionConstraint: '1.2.3',
    installedVersion: '1.2.3',
    installedReleaseId: 'release-1',
    acceptedPermissions: [],
    status: 'enabled',
    installedAt: '2026-07-26T00:00:00Z',
    updatedAt: '2026-07-26T00:00:00Z',
  };
  const client = new NekiroApiClient({
    baseUrl: 'https://api.example.test',
    token: 'owner-token',
    fetchImpl: async (input, init) => {
      requests.push({url: String(input), init});
      return new Response(JSON.stringify(installation), {status: 201, headers: {'Content-Type': 'application/json'}});
    },
  });
  const result = await client.installAgent('workspace.alpha', {agentId: 'agent.echo', versionConstraint: '1.2.3', acceptedPermissions: []});
  assert.equal(result.installedReleaseId, 'release-1');
  assert.equal(requests[0]?.url, 'https://api.example.test/v3/workspaces/workspace.alpha/installations');
  assert.deepEqual(JSON.parse(String(requests[0]?.init?.body)), {agentId: 'agent.echo', versionConstraint: '1.2.3', acceptedPermissions: []});
});

test('NekiroApiClient rejects Installation lifecycle responses that change immutable pins', async () => {
  const initial = {
    installationId: 'installation-1', workspaceId: 'workspace.alpha', agentId: 'agent.echo', versionConstraint: '^1.0.0', installedVersion: '1.2.3', installedReleaseId: 'release-1', acceptedPermissions: [], status: 'enabled', installedAt: '2026-07-26T00:00:00Z', updatedAt: '2026-07-26T00:00:00Z',
  };
  const mutations = [
    {...initial, versionConstraint: '^1.1.0'},
    {...initial, installedVersion: '1.3.0'},
    {...initial, acceptedPermissions: ['READ_LOGS']},
    {...initial, installedReleaseId: 'release-2'},
  ];
  for (const operation of ['update', 'uninstall'] as const) {
    for (const mutated of mutations) {
      let call = 0;
      const client = new NekiroApiClient({
        baseUrl: 'https://api.example.test',
        token: 'owner-token',
        fetchImpl: async () => new Response(JSON.stringify(call++ === 0 ? initial : mutated), {status: 200, headers: {'Content-Type': 'application/json'}}),
      });
      const request = operation === 'update'
        ? client.updateInstallation('workspace.alpha', 'installation-1', 'disabled')
        : client.uninstallAgent('workspace.alpha', 'installation-1');
      await assert.rejects(() => request, /immutable pin fields/);
    }
  }
});

test('trusted Installation validation preserves the exact request and Release identity', () => {
  const release = trustedRelease() as unknown as AgentRelease;
  const installation = {
    installationId: 'installation-1', workspaceId: 'workspace.alpha', agentId: 'agent.echo',
    versionConstraint: '1.2.3', installedVersion: '1.2.3', installedReleaseId: 'release-1',
    acceptedPermissions: ['READ_LOGS', 'WRITE_STATE'], status: 'enabled' as const, installedAt: '2026-07-26T00:00:00Z', updatedAt: '2026-07-26T00:00:00Z',
  };
  const expected = {
    workspaceId: 'workspace.alpha',
    agentId: 'agent.echo',
    versionConstraint: '1.2.3',
    acceptedPermissions: ['READ_LOGS', 'WRITE_STATE'],
  };
  assert.doesNotThrow(() => validateTrustedInstallation(installation, release, expected));

  const mutations = [
    {...installation, workspaceId: 'workspace.other'},
    {...installation, agentId: 'agent.other'},
    {...installation, versionConstraint: '^1.2.0'},
    {...installation, installedVersion: '1.2.4'},
    {...installation, installedReleaseId: 'release-2'},
    {...installation, acceptedPermissions: ['WRITE_STATE', 'READ_LOGS']},
    {...installation, acceptedPermissions: ['READ_LOGS']},
    {...installation, status: 'disabled' as const},
  ];
  for (const mutated of mutations) {
    assert.throws(() => validateTrustedInstallation(mutated, release, expected), /Release identity/);
  }
});

test('provider and Workspace-owner clients keep bearer contexts separate', async () => {
  const authorization: string[] = [];
  const response = trustedResponse(trustedRelease(), 200);
  const providerClient = new NekiroApiClient({baseUrl: 'https://api.example.test', token: 'provider-token', fetchImpl: async (_input, init) => { authorization.push(new Headers(init?.headers).get('Authorization') ?? ''); return response.clone(); }});
  const ownerClient = new NekiroApiClient({baseUrl: 'https://api.example.test', token: 'owner-token', fetchImpl: async (_input, init) => { authorization.push(new Headers(init?.headers).get('Authorization') ?? ''); return response.clone(); }});
  await providerClient.getAgentRelease('release-1');
  await ownerClient.getAgentRelease('release-1');
  assert.deepEqual(authorization, ['Bearer provider-token', 'Bearer owner-token']);
});

test('NekiroApiClient constructs a strict v4 JSON invocation request', async () => {
  const requests: Array<{url: string; init?: RequestInit}> = [];
  const client = new NekiroApiClient({
    baseUrl: 'https://api.example.test',
    token: 'exact-token',
    fetchImpl: async (input, init) => {
      requests.push({url: String(input), init});
      return new Response(JSON.stringify({schemaVersion: '1', invocationId: 'inv-1', rootTaskId: 'task-1', traceId: 'trace-1', status: 'succeeded', result: {ok: true}}), {status: 200, headers: {'Content-Type': 'application/json'}});
    },
  });
  const result = await client.invoke('workspace.alpha', {agentId: 'runtime.echo', capability: 'runtime.echo', input: {message: 'hello'}, stream: false});
  assert.deepEqual(result.result, {ok: true});
  assert.equal(requests[0]?.url, 'https://api.example.test/v4/workspaces/workspace.alpha/invocations');
  assert.deepEqual(JSON.parse(String(requests[0]?.init?.body)), {agentId: 'runtime.echo', capability: 'runtime.echo', input: {message: 'hello'}, stream: false});
});

test('NekiroApiClient preserves correlated Platform Error v4 fields', async () => {
  const client = new NekiroApiClient({
    baseUrl: 'https://api.example.test',
    token: 'test-token',
    fetchImpl: async () => new Response(JSON.stringify({code: 'TIMEOUT', message: 'The invocation timed out.', traceId: 'trace-1', invocationId: 'inv-1', rootTaskId: 'task-1'}), {status: 504, headers: {'Content-Type': 'application/json'}}),
  });
  await assert.rejects(() => client.invoke('workspace.alpha', {agentId: 'runtime.echo', capability: 'runtime.echo', input: {}, stream: false}), (error: unknown) => {
    assert.ok(error instanceof NekiroApiError);
    assert.equal(error.code, 'TIMEOUT');
    assert.equal(error.traceId, 'trace-1');
    assert.equal(error.invocationId, 'inv-1');
    assert.equal(error.rootTaskId, 'task-1');
    return true;
  });
});

test('NekiroApiClient reads Workspace-scoped v4 Invocation and Trace paths', async () => {
  const requests: string[] = [];
  const record = {invocationId: 'inv-1', rootTaskId: 'task-1', traceId: 'trace-1', caller: {type: 'user', id: 'owner-a'}, workspaceId: 'workspace.alpha', targetAgentId: 'runtime.echo', agentCardVersion: '1.0.0', capability: 'runtime.echo', status: 'pending', createdAt: '2026-07-21T00:00:00Z', updatedAt: '2026-07-21T00:00:00Z'};
  const event = {schemaVersion: '0.3', eventId: 'evt-1', sequence: 0, occurredAt: '2026-07-21T00:00:00Z', type: 'created', status: 'pending', invocationId: 'inv-1', rootTaskId: 'task-1', traceId: 'trace-1', caller: {type: 'user', id: 'owner-a'}, workspaceId: 'workspace.alpha', targetAgentId: 'runtime.echo', agentCardVersion: '1.0.0', capability: 'runtime.echo'};
  const client = new NekiroApiClient({baseUrl: 'https://api.example.test', token: 'test-token', fetchImpl: async (input) => {
    requests.push(String(input));
    return new Response(JSON.stringify(String(input).includes('/traces/') ? {traceId: 'trace-1', invocations: [record]} : {invocation: record, events: [event]}), {status: 200, headers: {'Content-Type': 'application/json'}});
  }});
  await client.getInvocation('workspace.alpha', 'inv-1');
  await client.getTrace('workspace.alpha', 'trace-1');
  assert.deepEqual(requests, ['https://api.example.test/v4/workspaces/workspace.alpha/invocations/inv-1', 'https://api.example.test/v4/workspaces/workspace.alpha/traces/trace-1']);
});

test('NekiroApiClient rejects Invocation Detail provenance changes', async () => {
  const cardDigest = 'aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa';
  const record = {invocationId: 'inv-1', rootTaskId: 'task-1', traceId: 'trace-1', caller: {type: 'user', id: 'owner-a'}, workspaceId: 'workspace.alpha', targetAgentId: 'runtime.echo', agentCardVersion: '1.0.0', agentReleaseId: 'release-1', agentCardDigest: cardDigest, capability: 'runtime.echo', status: 'pending', createdAt: '2026-07-21T00:00:00Z', updatedAt: '2026-07-21T00:00:00Z'};
  const event = {schemaVersion: '0.3', eventId: 'evt-1', sequence: 0, occurredAt: '2026-07-21T00:00:00Z', type: 'created', status: 'pending', invocationId: 'inv-1', rootTaskId: 'task-1', traceId: 'trace-1', caller: {type: 'user', id: 'owner-a'}, workspaceId: 'workspace.alpha', targetAgentId: 'runtime.echo', agentCardVersion: '1.0.0', agentReleaseId: 'release-1', agentCardDigest: 'bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb', capability: 'runtime.echo'};
  const client = new NekiroApiClient({baseUrl: 'https://api.example.test', token: 'test-token', fetchImpl: async () => new Response(JSON.stringify({invocation: record, events: [event]}), {status: 200, headers: {'Content-Type': 'application/json'}})});
  await assert.rejects(() => client.getInvocation('workspace.alpha', 'inv-1'), /Invocation Detail event correlation is invalid/);
});

test('NekiroApiClient validates ordered SSE events and requires a terminal event', async () => {
  const accepted = {schemaVersion: '2', sequence: 0, type: 'accepted', status: 'pending', invocationId: 'inv-1', rootTaskId: 'task-1', traceId: 'trace-1'};
  const completed = {schemaVersion: '2', sequence: 1, type: 'completed', status: 'succeeded', invocationId: 'inv-1', rootTaskId: 'task-1', traceId: 'trace-1'};
  const body = `data: ${JSON.stringify(accepted)}\n\ndata: ${JSON.stringify(completed)}\n\n`;
  const client = new NekiroApiClient({baseUrl: 'https://api.example.test', token: 'test-token', fetchImpl: async () => new Response(body, {status: 200, headers: {'Content-Type': 'text/event-stream'}})});
  const seen: number[] = [];
  const events = await client.invokeStream('workspace.alpha', {agentId: 'runtime.echo', capability: 'runtime.echo', input: {}}, (event) => seen.push(event.sequence));
  assert.deepEqual(seen, [0, 1]);
  assert.equal(events[1]?.type, 'completed');

  const interrupted = new NekiroApiClient({baseUrl: 'https://api.example.test', token: 'test-token', fetchImpl: async () => new Response(`data: ${JSON.stringify(accepted)}\n\n`, {status: 200, headers: {'Content-Type': 'text/event-stream'}})});
  await assert.rejects(() => interrupted.invokeStream('workspace.alpha', {agentId: 'runtime.echo', capability: 'runtime.echo', input: {}}), /ended before a terminal event/);
});

test('NekiroApiClient rejects SSE gaps, correlation changes, and mismatched terminal errors', async () => {
  const accepted = {schemaVersion: '2', sequence: 0, type: 'accepted', status: 'pending', invocationId: 'inv-1', rootTaskId: 'task-1', traceId: 'trace-1'};
  const gap = {schemaVersion: '2', sequence: 2, type: 'completed', status: 'succeeded', invocationId: 'inv-1', rootTaskId: 'task-1', traceId: 'trace-1'};
  const gapClient = new NekiroApiClient({baseUrl: 'https://api.example.test', token: 'test-token', fetchImpl: async () => new Response(`data: ${JSON.stringify(accepted)}\n\ndata: ${JSON.stringify(gap)}\n\n`, {status: 200, headers: {'Content-Type': 'text/event-stream'}})});
  await assert.rejects(() => gapClient.invokeStream('workspace.alpha', {agentId: 'runtime.echo', capability: 'runtime.echo', input: {}}), /sequence is not contiguous/);

  const changed = {...gap, sequence: 1, traceId: 'trace-other'};
  const changedClient = new NekiroApiClient({baseUrl: 'https://api.example.test', token: 'test-token', fetchImpl: async () => new Response(`data: ${JSON.stringify(accepted)}\n\ndata: ${JSON.stringify(changed)}\n\n`, {status: 200, headers: {'Content-Type': 'text/event-stream'}})});
  await assert.rejects(() => changedClient.invokeStream('workspace.alpha', {agentId: 'runtime.echo', capability: 'runtime.echo', input: {}}), /correlation changed/);

  const failed = {schemaVersion: '2', sequence: 1, type: 'failed', status: 'failed', invocationId: 'inv-1', rootTaskId: 'task-1', traceId: 'trace-1', error: {code: 'TIMEOUT', message: 'The invocation timed out.', traceId: 'trace-1', invocationId: 'inv-1', rootTaskId: 'task-1'}};
  const failedClient = new NekiroApiClient({baseUrl: 'https://api.example.test', token: 'test-token', fetchImpl: async () => new Response(`data: ${JSON.stringify(accepted)}\n\ndata: ${JSON.stringify(failed)}\n\n`, {status: 200, headers: {'Content-Type': 'text/event-stream'}})});
  await assert.rejects(() => failedClient.invokeStream('workspace.alpha', {agentId: 'runtime.echo', capability: 'runtime.echo', input: {}}), /failed stream event error code/);
});

test('NekiroApiClient accepts standard SSE comments, fields, and multiline data framing', async () => {
  const accepted = {schemaVersion: '2', sequence: 0, type: 'accepted', status: 'pending', invocationId: 'inv-1', rootTaskId: 'task-1', traceId: 'trace-1'};
  const completed = {schemaVersion: '2', sequence: 1, type: 'completed', status: 'succeeded', invocationId: 'inv-1', rootTaskId: 'task-1', traceId: 'trace-1'};
  const completedJSON = JSON.stringify(completed);
  const splitAt = completedJSON.indexOf(',"rootTaskId"');
  const body = `: keep-alive\nretry: 1000\nevent: result\ndata: ${JSON.stringify(accepted)}\n\ndata: ${completedJSON.slice(0, splitAt)}\ndata: ${completedJSON.slice(splitAt)}\n\n`;
  const client = new NekiroApiClient({baseUrl: 'https://api.example.test', token: 'test-token', fetchImpl: async () => new Response(body, {status: 200, headers: {'Content-Type': 'text/event-stream'}})});
  const events = await client.invokeStream('workspace.alpha', {agentId: 'runtime.echo', capability: 'runtime.echo', input: {}});
  assert.deepEqual(events.map((event) => event.type), ['accepted', 'completed']);
});

test('NekiroApiClient rejects omitted Installation limits instead of inventing one', () => {
  const client = new NekiroApiClient({baseUrl: 'https://api.example.test', token: 'test-token'});
  assert.throws(() => client.listInstallations('workspace.alpha', undefined as never), /limit must be an integer/);
});

test('NekiroApiClient rejects missing or whitespace bearer configuration', () => {
  assert.throws(() => new NekiroApiClient({baseUrl: 'https://api.example.test', token: ''}), /bearer token is required/);
  assert.throws(() => new NekiroApiClient({baseUrl: 'https://api.example.test', token: ' token'}), /must not contain whitespace/);
  assert.throws(() => new NekiroApiClient({baseUrl: 'https://api.example.test/v1', token: 'test-token'}), /base URL is invalid/);
  assert.throws(() => new NekiroApiClient({baseUrl: 'https://localhost', token: 'test-token'}), /base URL is invalid/);
  assert.throws(() => new NekiroApiClient({baseUrl: 'https://192.0.2.10', token: 'test-token'}), /base URL is invalid/);
});

test('NekiroApiClient rejects an empty JSON success body', async () => {
  const client = new NekiroApiClient({baseUrl: 'https://api.example.test', token: 'test-token', fetchImpl: async () => new Response('', {status: 200})});
  await assert.rejects(() => client.searchAgents(), /invalid JSON success response/);
});

test('NekiroApiClient rejects Gateway redirects for regular requests', async () => {
  let redirect: RequestRedirect | undefined;
  const client = new NekiroApiClient({
    baseUrl: 'https://api.example.test',
    token: 'test-token',
    fetchImpl: async (_input, init) => {
      redirect = init?.redirect;
      return new Response(JSON.stringify({items: []}), {status: 200, headers: {'Content-Type': 'application/json'}});
    },
  });
  await client.searchAgents();
  assert.equal(redirect, 'error');
});

test('NekiroApiClient constructs every Trusted Publication Gateway route without proof bodies', async () => {
  const requests: Array<{url: string; init?: RequestInit}> = [];
  const binding = trustedBinding();
  const challenge = trustedChallenge();
  const release = trustedRelease();
  const responses = [binding, binding, challenge, binding, release, release, release, release, release, release];
  const statuses = [201, 200, 201, 200, 201, 200, 200, 200, 200, 200];
  const client = new NekiroApiClient({
    baseUrl: 'https://api.example.test/',
    token: 'exact-token',
    fetchImpl: async (input, init) => {
      requests.push({url: String(input), init});
      const value = responses.shift();
      if (!value) throw new Error('unexpected request count');
      const status = statuses.shift();
      if (!status) throw new Error('unexpected response count');
      return trustedResponse(value, status);
    },
  });

  await client.createEndpointBinding('provider.main', 'agent.echo', {endpoint: 'https://agent.example/a2a', method: 'http_well_known', version: '1.2.3'});
  await client.getEndpointBinding('provider.main', 'binding-1');
  await client.createVerificationChallenge('provider.main', 'binding-1');
  await client.completeVerificationChallenge('provider.main', 'binding-1', 'challenge-1');
  await client.createAgentRelease('provider.main', 'agent.echo', {version: '1.2.3', endpointBindingId: 'binding-1'});
  await client.getAgentRelease('release-1');
  await client.verifyAgentRelease('release-1');
  await client.publishAgentRelease('release-1');
  await client.suspendAgentRelease('release-1');
  await client.revokeAgentRelease('release-1');

  assert.deepEqual(requests.map((request) => request.url), [
    'https://api.example.test/v4/providers/provider.main/agents/agent.echo/endpoint-bindings',
    'https://api.example.test/v4/providers/provider.main/endpoint-bindings/binding-1',
    'https://api.example.test/v4/providers/provider.main/endpoint-bindings/binding-1/challenges',
    'https://api.example.test/v4/providers/provider.main/endpoint-bindings/binding-1/challenges/challenge-1/complete',
    'https://api.example.test/v4/providers/provider.main/agents/agent.echo/releases',
    'https://api.example.test/v4/releases/release-1',
    'https://api.example.test/v4/releases/release-1/verify',
    'https://api.example.test/v4/releases/release-1/publish',
    'https://api.example.test/v4/releases/release-1/suspend',
    'https://api.example.test/v4/releases/release-1/revoke',
  ]);
  assert.deepEqual(requests.map((request) => request.init?.method), ['POST', undefined, 'POST', 'POST', 'POST', undefined, 'POST', 'POST', 'POST', 'POST']);
  assert.deepEqual(JSON.parse(String(requests[0]?.init?.body)), {endpoint: 'https://agent.example/a2a', method: 'http_well_known', version: '1.2.3'});
  assert.deepEqual(JSON.parse(String(requests[4]?.init?.body)), {version: '1.2.3', endpointBindingId: 'binding-1'});
  assert.equal(requests[2]?.init?.body, undefined);
  assert.equal(requests[3]?.init?.body, undefined);
  assert.equal(requests[5]?.init?.body, undefined);
  assert.equal(requests[2]?.init?.redirect, 'error');
  for (const request of requests) {
    const headers = new Headers(request.init?.headers);
    assert.equal(headers.get('Authorization'), 'Bearer exact-token');
  }
  assert.equal(requests.some((request) => String(request.init?.body ?? '').includes(String(challenge.proof))), false);
});

test('Trusted Publication errors preserve status, code, body trace, and optional matching header', async () => {
  const error = {code: 'WRONG_PROOF', message: 'The verification proof is incorrect.', traceId: 'trace-trust-1'};
  const client = new NekiroApiClient({
    baseUrl: 'https://api.example.test',
    token: 'test-token',
    fetchImpl: async () => trustedResponse(error, 400, {'x-nek-trace-id': 'trace-trust-1'}),
  });

  await assert.rejects(() => client.getAgentRelease('release-1'), (value: unknown) => {
    assert.ok(value instanceof NekiroApiError);
    assert.equal(value.status, 400);
    assert.equal(value.code, 'WRONG_PROOF');
    assert.equal(value.traceId, 'trace-trust-1');
    return true;
  });
});

test('Trusted Publication accepts an absent trace header but rejects a mismatched header', async () => {
  const error = {code: 'CHALLENGE_EXPIRED', message: 'The verification challenge expired.', traceId: 'trace-trust-2'};
  const absentHeader = new NekiroApiClient({
    baseUrl: 'https://api.example.test',
    token: 'test-token',
    fetchImpl: async () => trustedResponse(error, 409),
  });
  await assert.rejects(() => absentHeader.getAgentRelease('release-1'), (value: unknown) => {
    assert.ok(value instanceof NekiroApiError);
    assert.equal(value.code, 'CHALLENGE_EXPIRED');
    assert.equal(value.traceId, 'trace-trust-2');
    return true;
  });

  const mismatchedHeader = new NekiroApiClient({
    baseUrl: 'https://api.example.test',
    token: 'test-token',
    fetchImpl: async () => trustedResponse(error, 409, {'x-nek-trace-id': 'trace-other'}),
  });
  await assert.rejects(() => mismatchedHeader.getAgentRelease('release-1'), /inconsistent trace correlation/);
});

test('Trusted Publication rejects malformed success relationships and unknown fields', async () => {
  const mismatchedBinding = {...trustedBinding(), agentId: 'agent.other'};
  const bindingClient = new NekiroApiClient({
    baseUrl: 'https://api.example.test',
    token: 'test-token',
    fetchImpl: async () => trustedResponse(mismatchedBinding, 201),
  });
  await assert.rejects(
    () => bindingClient.createEndpointBinding('provider.main', 'agent.echo', {endpoint: 'https://agent.example/a2a', method: 'http_well_known', version: '1.2.3'}),
    /invalid response/,
  );

  const unknownRelease = {...trustedRelease(), unknown: true};
  const releaseClient = new NekiroApiClient({
    baseUrl: 'https://api.example.test',
    token: 'test-token',
    fetchImpl: async () => trustedResponse(unknownRelease, 200),
  });
  await assert.rejects(() => releaseClient.getAgentRelease('release-1'), /invalid response/);
});

test('Trusted Publication requires proof evidence for verified and published states', async () => {
  const incompleteBinding = {...trustedBinding()};
  delete incompleteBinding.verificationEvidenceDigest;
  const bindingClient = new NekiroApiClient({
    baseUrl: 'https://api.example.test',
    token: 'test-token',
    fetchImpl: async () => trustedResponse(incompleteBinding, 200),
  });
  await assert.rejects(() => bindingClient.getEndpointBinding('provider.main', 'binding-1'), /invalid response/);

  const revokedBinding: Record<string, unknown> = {...trustedBinding(), verificationStatus: 'revoked', revokedAt: '2026-07-26T00:00:03Z'};
  delete revokedBinding.verificationEvidenceDigest;
  delete revokedBinding.verifiedAt;
  const revokedBindingClient = new NekiroApiClient({
    baseUrl: 'https://api.example.test',
    token: 'test-token',
    fetchImpl: async () => trustedResponse(revokedBinding, 200),
  });
  assert.equal((await revokedBindingClient.getEndpointBinding('provider.main', 'binding-1')).verificationStatus, 'revoked');

  const missingRevokedAt = {...revokedBinding};
  delete missingRevokedAt.revokedAt;
  const missingRevokedAtClient = new NekiroApiClient({
    baseUrl: 'https://api.example.test',
    token: 'test-token',
    fetchImpl: async () => trustedResponse(missingRevokedAt, 200),
  });
  await assert.rejects(() => missingRevokedAtClient.getEndpointBinding('provider.main', 'binding-1'), /invalid response/);

  const pendingBinding: Record<string, unknown> = {...trustedBinding(), verificationStatus: 'pending'};
  delete pendingBinding.verificationEvidenceDigest;
  delete pendingBinding.verifiedAt;
  const pendingBindingClient = new NekiroApiClient({
    baseUrl: 'https://api.example.test',
    token: 'test-token',
    fetchImpl: async () => trustedResponse(pendingBinding, 200),
  });
  assert.equal((await pendingBindingClient.getEndpointBinding('provider.main', 'binding-1')).verificationStatus, 'pending');

  const incompleteRelease = {...trustedRelease()};
  delete incompleteRelease.publishedAt;
  const releaseClient = new NekiroApiClient({
    baseUrl: 'https://api.example.test',
    token: 'test-token',
    fetchImpl: async () => trustedResponse(incompleteRelease, 200),
  });
  await assert.rejects(() => releaseClient.getAgentRelease('release-1'), /invalid response/);

  const missingVerifiedAt = {...trustedRelease()};
  delete missingVerifiedAt.verifiedAt;
  const missingVerifiedAtClient = new NekiroApiClient({
    baseUrl: 'https://api.example.test',
    token: 'test-token',
    fetchImpl: async () => trustedResponse(missingVerifiedAt, 200),
  });
  await assert.rejects(() => missingVerifiedAtClient.getAgentRelease('release-1'), /invalid response/);

  for (const state of ['draft', 'pending_verification'] as const) {
    const validEarlyRelease: Record<string, unknown> = {...trustedRelease(), state};
    delete validEarlyRelease.verificationEvidenceDigest;
    delete validEarlyRelease.verifiedAt;
    delete validEarlyRelease.publishedAt;
    const earlyClient = new NekiroApiClient({
      baseUrl: 'https://api.example.test',
      token: 'test-token',
      fetchImpl: async () => trustedResponse(validEarlyRelease, 200),
    });
    assert.equal((await earlyClient.getAgentRelease('release-1')).state, state);

    const invalidEarlyRelease = {...validEarlyRelease, verificationEvidenceDigest: '0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef'};
    const invalidEarlyClient = new NekiroApiClient({
      baseUrl: 'https://api.example.test',
      token: 'test-token',
      fetchImpl: async () => trustedResponse(invalidEarlyRelease, 200),
    });
    await assert.rejects(() => invalidEarlyClient.getAgentRelease('release-1'), /invalid response/);
  }

  const publishedWithSuspensionTime = {...trustedRelease(), suspendedAt: '2026-07-26T00:00:03Z'};
  const inconsistentPublishedClient = new NekiroApiClient({
    baseUrl: 'https://api.example.test',
    token: 'test-token',
    fetchImpl: async () => trustedResponse(publishedWithSuspensionTime, 200),
  });
  await assert.rejects(() => inconsistentPublishedClient.getAgentRelease('release-1'), /invalid response/);

  for (const state of ['suspended', 'revoked'] as const) {
    const validTransition: Record<string, unknown> = {
      ...trustedRelease(),
      state,
      suspendedAt: '2026-07-26T00:00:03Z',
      revokedAt: state === 'revoked' ? '2026-07-26T00:00:04Z' : undefined,
    };
    if (state === 'suspended') delete validTransition.revokedAt;
    const validTransitionClient = new NekiroApiClient({
      baseUrl: 'https://api.example.test',
      token: 'test-token',
      fetchImpl: async () => trustedResponse(validTransition, 200),
    });
    assert.equal((await validTransitionClient.getAgentRelease('release-1')).state, state);

    const incompleteTransition: Record<string, unknown> = {...trustedRelease(), state};
    delete incompleteTransition.verificationEvidenceDigest;
    const transitionClient = new NekiroApiClient({
      baseUrl: 'https://api.example.test',
      token: 'test-token',
      fetchImpl: async () => trustedResponse(incompleteTransition, 200),
    });
    await assert.rejects(() => transitionClient.getAgentRelease('release-1'), /invalid response/);

    const timestampField = state === 'suspended' ? 'suspendedAt' : 'revokedAt';
    const incompleteTimestamp: Record<string, unknown> = {...trustedRelease(), state};
    delete incompleteTimestamp[timestampField];
    const timestampClient = new NekiroApiClient({
      baseUrl: 'https://api.example.test',
      token: 'test-token',
      fetchImpl: async () => trustedResponse(incompleteTimestamp, 200),
    });
    await assert.rejects(() => timestampClient.getAgentRelease('release-1'), /invalid response/);
  }

  const revokedFromVerified: Record<string, unknown> = {...trustedRelease(), state: 'revoked', revokedAt: '2026-07-26T00:00:04Z'};
  delete revokedFromVerified.publishedAt;
  const revokedFromVerifiedClient = new NekiroApiClient({
    baseUrl: 'https://api.example.test',
    token: 'test-token',
    fetchImpl: async () => trustedResponse(revokedFromVerified, 200),
  });
  assert.equal((await revokedFromVerifiedClient.getAgentRelease('release-1')).state, 'revoked');
});

test('Trusted Publication enforces operation-specific success status and strict endpoint data', async () => {
  const wrongStatusClient = new NekiroApiClient({
    baseUrl: 'https://api.example.test',
    token: 'test-token',
    fetchImpl: async () => trustedResponse(trustedBinding(), 200),
  });
  await assert.rejects(
    () => wrongStatusClient.createEndpointBinding('provider.main', 'agent.echo', {endpoint: 'https://agent.example/a2a', method: 'http_well_known', version: '1.2.3'}),
    /unexpected HTTP status/,
  );

  const userInfoClient = new NekiroApiClient({
    baseUrl: 'https://api.example.test',
    token: 'test-token',
    fetchImpl: async () => trustedResponse(trustedBinding(), 201),
  });
  assert.throws(
    () => userInfoClient.createEndpointBinding('provider.main', 'agent.echo', {endpoint: 'https://user:secret@agent.example/a2a', method: 'http_well_known', version: '1.2.3'}),
    /userinfo|HTTP\(S\) URI/,
  );
  assert.throws(
    () => userInfoClient.createEndpointBinding('provider.main', 'agent.echo', {endpoint: 'https://@agent.example/a2a', method: 'http_well_known', version: '1.2.3'}),
    /userinfo|HTTP\(S\) URI/,
  );
  assert.throws(
    () => userInfoClient.createEndpointBinding('provider.main', 'agent.echo', {endpoint: ' https://agent.example/a2a', method: 'http_well_known', version: '1.2.3'}),
    /whitespace/,
  );

  const invalidDateClient = new NekiroApiClient({
    baseUrl: 'https://api.example.test',
    token: 'test-token',
    fetchImpl: async () => trustedResponse({...trustedBinding(), updatedAt: '2026-02-31T00:00:00Z'}, 200),
  });
  await assert.rejects(() => invalidDateClient.getEndpointBinding('provider.main', 'binding-1'), /invalid response/);
});

test('Trusted Publication transport errors do not expose the underlying error message', async () => {
  const client = new NekiroApiClient({
    baseUrl: 'https://api.example.test',
    token: 'test-token',
    fetchImpl: async () => { throw new Error('internal socket detail and secret-value'); },
  });
  await assert.rejects(() => client.getAgentRelease('release-1'), (value: unknown) => {
    assert.ok(value instanceof NekiroApiError);
    assert.equal(value.code, 'NETWORK_ERROR');
    assert.equal(value.message, 'NeKiro API request failed.');
    assert.equal(value.message.includes('secret-value'), false);
    return true;
  });
});

test('Trusted Publication malformed error bodies return a safe validation error after one read', async () => {
  const client = new NekiroApiClient({
    baseUrl: 'https://api.example.test',
    token: 'test-token',
    fetchImpl: async () => new Response('', {status: 500, headers: {'Content-Type': 'application/json'}}),
  });
  await assert.rejects(() => client.getAgentRelease('release-1'), (value: unknown) => {
    assert.ok(value instanceof NekiroApiError);
    assert.equal(value.code, 'INVALID_RESPONSE');
    assert.equal(value.message, 'NeKiro Trusted Publication returned an invalid error response.');
    return true;
  });
});

function trustedResponse(value: unknown, status: number, extraHeaders: Record<string, string> = {}): Response {
  return new Response(JSON.stringify(value), {
    status,
    headers: {'Content-Type': 'application/json', ...extraHeaders},
  });
}

function catalogCard(): AgentCardV02 {
  return {
    schemaVersion: '0.2',
    agentId: 'agent.echo',
    name: 'Echo Agent',
    description: 'Echoes structured input.',
    owner: {id: 'provider.main', displayName: 'Provider Main'},
    version: '1.2.3',
    protocol: {type: 'a2a', version: '0.3.0', transport: 'JSONRPC', endpoint: 'https://agent.example/a2a'},
    skills: [{id: 'runtime.echo', name: 'Echo', description: 'Echo input.', inputSchema: {type: 'object'}, outputSchema: {type: 'object'}, requiredPermissions: ['READ_LOGS']}],
    authentication: {type: 'none'},
    permissions: [{id: 'READ_LOGS', description: 'Read logs.'}],
    limits,
  };
}

function trustedBinding(): Record<string, unknown> {
  return {
    bindingId: 'binding-1',
    providerId: 'provider.main',
    agentId: 'agent.echo',
    agentCardVersion: '1.2.3',
    endpoint: 'https://agent.example/a2a',
    verificationMethod: 'http_well_known',
    verificationStatus: 'verified',
    verificationEvidenceDigest: '0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef',
    createdAt: '2026-07-26T00:00:00Z',
    updatedAt: '2026-07-26T00:00:01Z',
    verifiedAt: '2026-07-26T00:00:01Z',
  };
}

function trustedChallenge(): Record<string, unknown> {
  return {
    challengeId: 'challenge-1',
    bindingId: 'binding-1',
    challengeUrl: 'https://agent.example/.well-known/nekiro/challenges/challenge-1',
    proof: 'one-time-proof',
    expiresAt: '2026-07-26T00:05:00Z',
  };
}

function trustedRelease(): Record<string, unknown> {
  return {
    releaseId: 'release-1',
    providerId: 'provider.main',
    agentId: 'agent.echo',
    agentCardVersion: '1.2.3',
    cardDigest: 'abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789',
    endpointBindingId: 'binding-1',
    endpointOrigin: 'https://agent.example',
    endpointPath: '/a2a',
    verificationMethod: 'http_well_known',
    verificationEvidenceDigest: '0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef',
    state: 'published',
    createdAt: '2026-07-26T00:00:00Z',
    updatedAt: '2026-07-26T00:00:02Z',
    verifiedAt: '2026-07-26T00:00:01Z',
    publishedAt: '2026-07-26T00:00:02Z',
  };
}
