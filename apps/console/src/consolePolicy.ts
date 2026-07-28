import type {Agent, Installation} from './types';
import type {AgentRelease, AgentReleaseState} from './api/nekiro';

export type ReleaseLifecycleAction = 'verify' | 'publish' | 'suspend' | 'revoke';

export function agentKey(agent: Pick<Agent, 'id' | 'version'>): string {
  return agent.id + '@' + agent.version;
}

export function isTrustedEnabledInstallation(installation: Installation): boolean {
  return installation.status === 'enabled' && typeof installation.installedReleaseId === 'string' && installation.installedReleaseId.length > 0;
}

export function matchesPublishedRelease(release: AgentRelease, agent: Pick<Agent, 'id' | 'version' | 'ownerId'>): boolean {
  return release.state === 'published'
    && release.providerId === agent.ownerId
    && release.agentId === agent.id
    && release.agentCardVersion === agent.version;
}

export function canReleaseAction(state: AgentReleaseState | undefined, action: ReleaseLifecycleAction): boolean {
  if (action === 'verify') return state === 'pending_verification';
  if (action === 'publish') return state === 'verified';
  if (action === 'suspend') return state === 'verified' || state === 'published';
  return state === 'verified' || state === 'published' || state === 'suspended';
}

export function nextRequestGeneration(current: number): number {
  return current + 1;
}

export function isCurrentRequest(generation: number, current: number): boolean {
  return generation === current;
}
