export type PublicationStatus = 'draft' | 'published' | 'disabled';
export type InstallationStatus = 'enabled' | 'disabled' | 'uninstalled';

export interface AgentPermissionSummary {
  id: string;
  description: string;
}

export interface Agent {
  id: string;
  name: string;
  version: string;
  owner: string;
  ownerId: string;
  description: string;
  tags: string[];
  status: PublicationStatus;
  schema: string;
  permissions: AgentPermissionSummary[];
  registeredAt: string;
  publishedAt?: string;
  publicAgentId?: string;
  publicUrl?: string;
}

export interface Workspace {
  workspaceId: string;
  ownerId: string;
  createdAt: string;
  updatedAt: string;
}

export interface Installation {
  installationId: string;
  workspaceId: string;
  agentId: string;
  versionConstraint: string;
  installedVersion: string;
  installedReleaseId?: string;
  acceptedPermissions: string[];
  status: InstallationStatus;
  installedAt: string;
  updatedAt: string;
  uninstalledAt?: string;
}

export interface PlatformErrorView {
  status: number;
  code?: string;
  message: string;
  traceId?: string;
  invocationId?: string;
  rootTaskId?: string;
}

export interface TraceNode {
  id: string;
  name: string;
  duration?: string;
  iconName: string;
  status?: 'SUCCESS' | 'TIMEOUT' | 'ERROR' | 'PENDING';
  children?: TraceNode[];
}

export interface LedgerEvent {
  id: string;
  time: string;
  title: string;
  subtitle?: string;
  badgeText?: string;
  badgeType?: 'success' | 'warning' | 'error' | 'default';
  details?: { label: string; value: string }[];
  errorBox?: { title: string; text: string; code: string };
}

export interface LogChunk {
  timestamp: string;
  type: 'info' | 'success' | 'warning' | 'error' | 'stream' | 'chunk';
  message: string;
  details?: string;
  chunkData?: string;
}
