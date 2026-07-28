import type {Agent, Installation, Workspace} from '../types';

export const DEMO_WORKSPACE: Workspace = {
  workspaceId: 'ws-prod-01',
  ownerId: 'team.platform',
  createdAt: '2026-06-02T09:14:00Z',
  updatedAt: '2026-07-18T16:42:00Z',
};

const echoCard = {
  schemaVersion: '0.2',
  agentId: 'runtime.echo',
  name: 'Runtime Echo',
  description: 'Echoes structured input through the A2A JSON-RPC profile. Used for connectivity probes and contract tests.',
  owner: {id: 'team.platform', displayName: 'Platform Team'},
  version: '1.0.0',
  protocol: {type: 'a2a', version: '0.3.0', transport: 'JSONRPC', endpoint: 'http://127.0.0.1:9000/a2a'},
  skills: [
    {id: 'runtime.echo', name: 'Runtime Echo', description: 'Echoes structured input.', inputSchema: {type: 'object'}, outputSchema: {type: 'object'}, requiredPermissions: []},
  ],
  authentication: {type: 'none'},
  permissions: [],
  limits: {timeoutMs: 30000, maxInputBytes: 1048576, maxOutputBytes: 1048576, streaming: true},
};

const analyzerCard = {
  ...echoCard,
  agentId: 'text.analyzer',
  name: 'Text Analyzer',
  version: '2.1.3',
  description: 'Runs lexical and semantic analysis over submitted documents, emitting structured annotations.',
  protocol: {...echoCard.protocol, endpoint: 'https://agents.nekiro.dev/text-analyzer/a2a'},
  skills: [
    {id: 'text.analyze', name: 'Analyze', description: 'Full-document analysis.', inputSchema: {type: 'object'}, outputSchema: {type: 'object'}, requiredPermissions: ['docs.read']},
    {id: 'text.summarize', name: 'Summarize', description: 'Abstractive summary.', inputSchema: {type: 'object'}, outputSchema: {type: 'object'}, requiredPermissions: ['docs.read']},
  ],
  authentication: {type: 'http_bearer'},
  permissions: [
    {id: 'docs.read', description: 'Read documents submitted for analysis.'},
    {id: 'metrics.write', description: 'Emit anonymized usage metrics.'},
  ],
};

const reviewCard = {
  ...echoCard,
  agentId: 'code.review',
  name: 'Code Review',
  version: '0.9.0',
  description: 'Reviews pull requests against repository policy and posts structured findings.',
  protocol: {...echoCard.protocol, endpoint: 'https://agents.nekiro.dev/code-review/a2a'},
  skills: [
    {id: 'review.diff', name: 'Review Diff', description: 'Review a pull request diff.', inputSchema: {type: 'object'}, outputSchema: {type: 'object'}, requiredPermissions: ['repo.read', 'pr.write']},
  ],
  authentication: {type: 'oauth2_client_credentials'},
  permissions: [
    {id: 'repo.read', description: 'Read repository contents and diffs.'},
    {id: 'pr.write', description: 'Post review comments to pull requests.'},
  ],
};

const deployCard = {
  ...echoCard,
  agentId: 'deploy.captain',
  name: 'Deploy Captain',
  version: '3.0.1',
  description: 'Orchestrates canary deployments and rollback decisions across staging and production.',
  skills: [
    {id: 'deploy.plan', name: 'Plan', description: 'Build a deploy plan.', inputSchema: {type: 'object'}, outputSchema: {type: 'object'}, requiredPermissions: ['deploy.execute']},
  ],
  authentication: {type: 'mutual_tls'},
  permissions: [{id: 'deploy.execute', description: 'Trigger deployments and rollbacks.'}],
};

const visionCard = {
  ...echoCard,
  agentId: 'vision.tagger',
  name: 'Vision Tagger',
  version: '1.4.0',
  description: 'Tags image assets with detected objects, scenes, and OCR text.',
  skills: [
    {id: 'vision.tag', name: 'Tag', description: 'Tag an image.', inputSchema: {type: 'object'}, outputSchema: {type: 'object'}, requiredPermissions: ['assets.read']},
  ],
  authentication: {type: 'api_key'},
  permissions: [{id: 'assets.read', description: 'Read image assets from the bucket.'}],
};

const ledgerCard = {
  ...echoCard,
  agentId: 'ledger.audit',
  name: 'Ledger Audit',
  version: '0.3.2',
  description: 'Cross-checks invocation lineage against ledger records and reports drift.',
  skills: [
    {id: 'ledger.verify', name: 'Verify', description: 'Verify lineage integrity.', inputSchema: {type: 'object'}, outputSchema: {type: 'object'}, requiredPermissions: ['ledger.read']},
  ],
  authentication: {type: 'http_bearer'},
  permissions: [{id: 'ledger.read', description: 'Read metadata-only ledger records.'}],
};

export const DEMO_AGENTS: Agent[] = [
  {id: 'runtime.echo', name: 'Runtime Echo', version: '1.0.0', owner: 'Platform Team', ownerId: 'team.platform', description: echoCard.description, tags: ['runtime.echo'], status: 'published', schema: JSON.stringify(echoCard, null, 2), permissions: [], registeredAt: '2026-06-02T09:20:00Z', publishedAt: '2026-06-02T10:05:00Z'},
  {id: 'text.analyzer', name: 'Text Analyzer', version: '2.1.3', owner: 'Platform Team', ownerId: 'team.platform', description: analyzerCard.description, tags: ['text.analyze', 'text.summarize'], status: 'published', schema: JSON.stringify(analyzerCard, null, 2), permissions: analyzerCard.permissions, registeredAt: '2026-06-11T13:00:00Z', publishedAt: '2026-06-12T08:30:00Z'},
  {id: 'code.review', name: 'Code Review', version: '0.9.0', owner: 'DevEx Guild', ownerId: 'guild.devex', description: reviewCard.description, tags: ['review.diff'], status: 'draft', schema: JSON.stringify(reviewCard, null, 2), permissions: reviewCard.permissions, registeredAt: '2026-07-15T17:45:00Z'},
  {id: 'deploy.captain', name: 'Deploy Captain', version: '3.0.1', owner: 'Release Ops', ownerId: 'team.release', description: deployCard.description, tags: ['deploy.plan'], status: 'published', schema: JSON.stringify(deployCard, null, 2), permissions: deployCard.permissions, registeredAt: '2026-05-20T11:00:00Z', publishedAt: '2026-05-21T09:12:00Z'},
  {id: 'vision.tagger', name: 'Vision Tagger', version: '1.4.0', owner: 'ML Platform', ownerId: 'team.ml', description: visionCard.description, tags: ['vision.tag'], status: 'published', schema: JSON.stringify(visionCard, null, 2), permissions: visionCard.permissions, registeredAt: '2026-06-28T15:20:00Z', publishedAt: '2026-06-29T10:00:00Z'},
  {id: 'ledger.audit', name: 'Ledger Audit', version: '0.3.2', owner: 'Platform Team', ownerId: 'team.platform', description: ledgerCard.description, tags: ['ledger.verify'], status: 'disabled', schema: JSON.stringify(ledgerCard, null, 2), permissions: ledgerCard.permissions, registeredAt: '2026-04-08T09:00:00Z', publishedAt: '2026-04-08T12:00:00Z'},
];

export const DEMO_INSTALLATIONS: Installation[] = [
  {installationId: 'ins-01JYX4K2M0', workspaceId: 'ws-prod-01', agentId: 'runtime.echo', versionConstraint: '1.0.0', installedVersion: '1.0.0', acceptedPermissions: [], status: 'enabled', installedAt: '2026-06-02T10:20:00Z', updatedAt: '2026-06-02T10:20:00Z'},
  {installationId: 'ins-01JZX9Q4TA', workspaceId: 'ws-prod-01', agentId: 'text.analyzer', versionConstraint: '^2.1.0', installedVersion: '2.1.3', acceptedPermissions: ['docs.read'], status: 'enabled', installedAt: '2026-06-12T09:00:00Z', updatedAt: '2026-07-01T14:32:00Z'},
  {installationId: 'ins-01K0B3R8WD', workspaceId: 'ws-prod-01', agentId: 'deploy.captain', versionConstraint: '3.0.1', installedVersion: '3.0.1', acceptedPermissions: ['deploy.execute'], status: 'disabled', installedAt: '2026-05-21T09:40:00Z', updatedAt: '2026-07-10T18:05:00Z'},
  {installationId: 'ins-01JYW7N1PE', workspaceId: 'ws-prod-01', agentId: 'ledger.audit', versionConstraint: '0.3.2', installedVersion: '0.3.2', acceptedPermissions: ['ledger.read'], status: 'uninstalled', installedAt: '2026-04-09T08:00:00Z', updatedAt: '2026-06-30T11:15:00Z', uninstalledAt: '2026-06-30T11:15:00Z'},
];

export function matchesAgent(agent: Agent, query: string): boolean {
  const q = query.trim().toLowerCase();
  if (!q) return true;
  return [agent.id, agent.name, agent.owner, agent.ownerId, agent.description, agent.version, agent.status, agent.tags.join(' ')]
    .join(' ')
    .toLowerCase()
    .includes(q);
}

export function shortDate(iso?: string): string {
  if (!iso) return '—';
  return iso.slice(0, 10);
}
