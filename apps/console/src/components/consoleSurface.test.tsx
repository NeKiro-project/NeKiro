import assert from 'node:assert/strict';
import test from 'node:test';
import {renderToStaticMarkup} from 'react-dom/server';

import type {AgentRelease, NekiroApiClient} from '../api/nekiro';
import InstallationsTab from './InstallationsTab';
import InvocationsTab from './InvocationsTab';
import TrustedPublicationTab from './TrustedPublicationTab';
import type {Agent, Installation, Workspace} from '../types';

const client = {} as NekiroApiClient;
const workspace: Workspace = {
  workspaceId: 'workspace.alpha',
  ownerId: 'owner.alpha',
  createdAt: '2026-07-26T00:00:00Z',
  updatedAt: '2026-07-26T00:00:00Z',
};

function agent(id: string, version: string, ownerId = 'provider.main', status: Agent['status'] = 'published'): Agent {
  return {
    id, name: id, version, owner: ownerId, ownerId, description: id, tags: ['runtime.echo'], status,
    schema: '{}', permissions: [], registeredAt: '2026-07-26T00:00:00Z',
  };
}

function installation(agentId: string, status: Installation['status'], installedReleaseId?: string): Installation {
  return {
    installationId: agentId + '-installation', workspaceId: workspace.workspaceId, agentId,
    versionConstraint: '1.2.3', installedVersion: '1.2.3', installedReleaseId,
    acceptedPermissions: [], status, installedAt: '2026-07-26T00:00:00Z', updatedAt: '2026-07-26T00:00:00Z',
  };
}

test('Installations surface keeps same-Agent versions distinct', () => {
  const markup = renderToStaticMarkup(
    <InstallationsTab
      workspace={workspace}
      agents={[agent('agent.echo', '1.0.0'), agent('agent.echo', '1.1.0')]}
      installations={[]}
      loading={false}
      error={null}
      searchQuery=""
      client={client}
      onInstallAgent={async (_agent: Agent, _release: AgentRelease, _permissions: string[]) => {}}
      onUpdateInstallation={async (_installation: Installation, _status: 'enabled' | 'disabled') => {}}
      onUninstall={async (_installation: Installation) => true}
      onRefresh={() => {}}
    />,
  );

  assert.match(markup, /agent\.echo@1\.0\.0/);
  assert.match(markup, /agent\.echo@1\.1\.0/);
});

test('Installations surface requires explicit Agent and permission authorization', () => {
  const markup = renderToStaticMarkup(
    <InstallationsTab
      workspace={workspace}
      agents={[{...agent('agent.secure', '1.0.0'), permissions: [{id: 'documents.read', description: 'Read documents.'}]}]}
      installations={[]}
      loading={false}
      error={null}
      searchQuery=""
      client={client}
      onInstallAgent={async (_agent: Agent, _release: AgentRelease, _permissions: string[]) => {}}
      onUpdateInstallation={async (_installation: Installation, _status: 'enabled' | 'disabled') => {}}
      onUninstall={async (_installation: Installation) => true}
      onRefresh={() => {}}
    />,
  );

  assert.match(markup, /Select a published Agent/);
  assert.doesNotMatch(markup, /documents\.read/);
  assert.doesNotMatch(markup, /checked/);
});

test('Invocation surface excludes enabled legacy Installations without Release identity', () => {
  const markup = renderToStaticMarkup(<InvocationsTab workspace={workspace} installations={[
    installation('legacy.echo', 'enabled'),
    installation('agent.echo', 'enabled', 'release-1'),
    installation('disabled.echo', 'disabled', 'release-2'),
  ]} client={client} />);

  assert.match(markup, /release-1/);
  assert.doesNotMatch(markup, /legacy\.echo/);
  assert.doesNotMatch(markup, /disabled\.echo/);
});

test('Trusted Publication surface only renders provider-owned Cards', () => {
  const markup = renderToStaticMarkup(
    <TrustedPublicationTab
      providerId="provider.main"
      client={client}
      agents={[agent('provider.agent', '1.0.0', 'provider.main'), agent('other.agent', '1.0.0', 'provider.other')]}
      draftAgents={[]}
      providerCatalogError={null}
      onRefresh={() => {}}
    />,
  );

  assert.match(markup, /provider\.agent/);
  assert.doesNotMatch(markup, /other\.agent/);
});
