import {satisfies as semverSatisfies, valid as semverValid, validRange as semverValidRange} from 'semver';

import type {Agent, Installation, InstallationStatus, PlatformErrorView, Workspace} from '../types';

export type PublicationStatus = 'draft' | 'published' | 'disabled';
export type AuthenticationType = 'none' | 'api_key' | 'http_bearer' | 'oauth2_client_credentials' | 'mutual_tls';
export type JsonObject = Record<string, unknown>;

export interface AgentSkill {
  id: string;
  name: string;
  description: string;
  inputSchema: JsonObject;
  outputSchema: JsonObject;
  requiredPermissions: string[];
}

export interface AgentPermission {
  id: string;
  description: string;
}

export interface AgentCardV02 {
  schemaVersion: '0.2';
  agentId: string;
  name: string;
  description: string;
  owner: {
    id: string;
    displayName: string;
  };
  version: string;
  protocol: {
    type: 'a2a';
    version: '0.3.0';
    transport: 'JSONRPC';
    endpoint: string;
  };
  skills: AgentSkill[];
  authentication: {
    type: AuthenticationType;
  };
  permissions: AgentPermission[];
  limits: {
    timeoutMs: number;
    maxInputBytes: number;
    maxOutputBytes: number;
    streaming: boolean;
  };
}

export interface CatalogEntry {
  card: AgentCardV02;
  publicationStatus: PublicationStatus;
  registeredAt: string;
  publishedAt?: string;
  publicAgentId?: string;
  publicUrl?: string;
}

export interface PublicAgentRelease {
  releaseId: string;
  agentId: string;
  name: string;
  description: string;
  owner: {id: string; displayName: string};
  agentCardVersion: string;
  cardDigest: string;
  publishedAt: string;
  authenticationType: AuthenticationType;
  skills: AgentSkill[];
  permissions: AgentPermission[];
  limits: AgentCardV02['limits'];
}

export interface PublicAgentShare {
  schemaVersion: '1';
  publicAgentId: string;
  publicUrl: string;
  registeredAt: string;
  availability: 'installable' | 'not_installable';
  releases: PublicAgentRelease[];
}

export interface CatalogSearchResponse {
  items: CatalogEntry[];
  nextCursor?: string;
}

export interface CatalogSearchParams {
  query?: string;
  capability?: string;
  ownerId?: string;
  limit?: number;
  cursor?: string;
}

export interface AgentCardInput {
  agentId: string;
  name: string;
  ownerId: string;
  ownerDisplayName: string;
  description: string;
  version: string;
  endpoint: string;
  authentication: AuthenticationType;
  permissions: AgentPermission[];
  capabilitiesJson: string;
  limits: AgentCardV02['limits'];
}

export interface InstallAgentRequest {
  agentId: string;
  versionConstraint: string;
  acceptedPermissions: string[];
}

export interface InstallationList {
  items: Installation[];
  nextCursor?: string;
}

export type TrustedPublicationErrorCode =
  | 'VALIDATION_ERROR' | 'UNAUTHENTICATED' | 'FORBIDDEN' | 'NOT_FOUND' | 'CONFLICT'
  | 'INVALID_ENDPOINT' | 'DISALLOWED_NETWORK' | 'ENDPOINT_UNAVAILABLE'
  | 'WRONG_PROOF' | 'CHALLENGE_EXPIRED' | 'CHALLENGE_REUSED'
  | 'REDIRECT_NOT_ALLOWED' | 'DEPENDENCY_ERROR' | 'INTERNAL_ERROR';

export type EndpointBindingVerificationStatus = 'pending' | 'verified' | 'failed' | 'revoked';
export type AgentReleaseState = 'draft' | 'pending_verification' | 'verified' | 'published' | 'suspended' | 'revoked';

export interface CreateEndpointBindingRequest {
  endpoint: string;
  method: 'http_well_known';
  version: string;
}

export interface EndpointBinding {
  bindingId: string;
  providerId: string;
  agentId: string;
  agentCardVersion: string;
  endpoint: string;
  verificationMethod: 'http_well_known';
  verificationStatus: EndpointBindingVerificationStatus;
  verificationFailureCode?: string;
  verificationEvidenceDigest?: string;
  createdAt: string;
  updatedAt: string;
  verifiedAt?: string;
  revokedAt?: string;
}

export interface VerificationChallenge {
  challengeId: string;
  bindingId: string;
  challengeUrl: string;
  proof: string;
  expiresAt: string;
}

export interface CreateAgentReleaseRequest {
  version: string;
  endpointBindingId: string;
}

export interface AgentRelease {
  releaseId: string;
  providerId: string;
  agentId: string;
  agentCardVersion: string;
  cardDigest: string;
  endpointBindingId: string;
  endpointOrigin: string;
  endpointPath: string;
  verificationMethod: 'http_well_known';
  verificationEvidenceDigest?: string;
  state: AgentReleaseState;
  createdAt: string;
  updatedAt: string;
  verifiedAt?: string;
  publishedAt?: string;
  suspendedAt?: string;
  revokedAt?: string;
}

export type PlatformErrorCode =
  | 'VALIDATION_ERROR' | 'UNAUTHENTICATED' | 'FORBIDDEN' | 'NOT_FOUND' | 'CONFLICT'
  | 'NOT_ACCEPTABLE' | 'PAYLOAD_TOO_LARGE' | 'AGENT_NOT_INSTALLED'
  | 'INSTALLATION_DISABLED' | 'AGENT_DISABLED' | 'AGENT_RELEASE_UNPUBLISHED'
  | 'AGENT_RELEASE_SUSPENDED' | 'AGENT_RELEASE_REVOKED' | 'CAPABILITY_NOT_ALLOWED'
  | 'ROUTE_NOT_FOUND' | 'AGENT_AUTH_UNSUPPORTED' | 'AGENT_RESPONSE_TOO_LARGE'
  | 'A2A_PROTOCOL_ERROR' | 'AGENT_UNAVAILABLE' | 'AGENT_EXECUTION_FAILED'
  | 'DEPENDENCY_ERROR' | 'TIMEOUT' | 'CANCELED' | 'INTERNAL_ERROR';

export interface PreCorrelationPlatformErrorV4 {
  code: PlatformErrorCode;
  message: string;
  traceId: string;
}

export interface CorrelatedPlatformErrorV4 extends PreCorrelationPlatformErrorV4 {
  invocationId: string;
  rootTaskId: string;
}

export type PlatformErrorV4 = PreCorrelationPlatformErrorV4 | CorrelatedPlatformErrorV4;

export interface InvocationRequestV4 {
  agentId: string;
  capability: string;
  input: JsonObject;
  stream: boolean;
}

export interface InvocationResultV1 {
  schemaVersion: '1';
  invocationId: string;
  rootTaskId: string;
  traceId: string;
  status: 'succeeded';
  result: unknown;
}

export type ResultStreamEventType = 'accepted' | 'chunk' | 'completed' | 'failed' | 'canceled' | 'timed_out';
export type InvocationResultStatus = 'pending' | 'running' | 'succeeded' | 'failed' | 'canceled' | 'timed_out';

export interface InvocationResultStreamEventV2 {
  schemaVersion: '2';
  sequence: number;
  type: ResultStreamEventType;
  status: InvocationResultStatus;
  invocationId: string;
  rootTaskId: string;
  traceId: string;
  chunkIndex?: number;
  chunk?: unknown;
  error?: CorrelatedPlatformErrorV4;
}

export type InvocationEventType = 'created' | 'routing' | 'started' | 'stream' | 'succeeded' | 'failed' | 'canceled' | 'timed_out';
export type InvocationEventStatus = 'pending' | 'routing' | 'running' | 'succeeded' | 'failed' | 'canceled' | 'timed_out';

export interface InvocationEventV03 {
  schemaVersion: '0.3';
  eventId: string;
  sequence: number;
  occurredAt: string;
  type: InvocationEventType;
  status: InvocationEventStatus;
  invocationId: string;
  rootTaskId: string;
  parentInvocationId?: string;
  traceId: string;
  caller: {type: 'user' | 'agent' | 'service'; id: string};
  workspaceId: string;
  targetAgentId: string;
  agentCardVersion: string;
  agentReleaseId?: string;
  agentCardDigest?: string;
  capability: string;
  chunkIndex?: number;
  chunkBytes?: number;
  latencyMs?: number;
  error?: CorrelatedPlatformErrorV4;
}

export interface InvocationRecordV4 {
  invocationId: string;
  rootTaskId: string;
  parentInvocationId?: string;
  traceId: string;
  caller: {type: 'user' | 'agent' | 'service'; id: string};
  workspaceId: string;
  targetAgentId: string;
  agentCardVersion: string;
  agentReleaseId?: string;
  agentCardDigest?: string;
  capability: string;
  status: InvocationEventStatus;
  latencyMs?: number;
  errorCode?: PlatformErrorCode;
  createdAt: string;
  updatedAt: string;
}

export interface InvocationDetailResponseV4 {
  invocation: InvocationRecordV4;
  events: InvocationEventV03[];
}

export interface TraceResponseV4 {
  traceId: string;
  invocations: InvocationRecordV4[];
}

export class NekiroApiError extends Error {
  readonly status: number;
  readonly code?: string;
  readonly traceId?: string;
  readonly invocationId?: string;
  readonly rootTaskId?: string;

  constructor(status: number, message: string, code?: string, traceId?: string, invocationId?: string, rootTaskId?: string) {
    super(message);
    this.name = 'NekiroApiError';
    this.status = status;
    this.code = code;
    this.traceId = traceId;
    this.invocationId = invocationId;
    this.rootTaskId = rootTaskId;
  }

  toView(): PlatformErrorView {
    return {
      status: this.status,
      code: this.code,
      message: this.message,
      traceId: this.traceId,
      invocationId: this.invocationId,
      rootTaskId: this.rootTaskId,
    };
  }
}

interface NekiroApiClientOptions {
  baseUrl: string;
  token?: string;
  anonymousOnly?: boolean;
  publicAgentOrigin?: string;
  fetchImpl?: typeof fetch;
}

export class NekiroApiClient {
  private readonly baseUrl: string;
  private readonly token: string;
  private readonly fetchImpl: typeof fetch;
  private readonly publicAgentOrigin: string;

  constructor(options: NekiroApiClientOptions) {
    if (typeof options.baseUrl !== 'string' || options.baseUrl === '' || options.baseUrl !== options.baseUrl.trim()) {
      throw new NekiroApiError(0, 'NeKiro Control Plane API base URL is required and must not contain surrounding whitespace.', 'CONFIGURATION_ERROR');
    }
    let parsedBaseUrl: URL;
    try {
      parsedBaseUrl = new URL(options.baseUrl);
    } catch {
      throw new NekiroApiError(0, 'NeKiro Control Plane API base URL is invalid.', 'CONFIGURATION_ERROR');
    }
    if (!['http:', 'https:'].includes(parsedBaseUrl.protocol)
      || parsedBaseUrl.username
      || parsedBaseUrl.password
      || parsedBaseUrl.search
      || parsedBaseUrl.hash
      || parsedBaseUrl.pathname !== '/'
      || parsedBaseUrl.hostname === 'localhost'
      || parsedBaseUrl.hostname.includes('*')
      || isIpHostname(parsedBaseUrl.hostname)) {
      throw new NekiroApiError(0, 'NeKiro Control Plane API base URL is invalid.', 'CONFIGURATION_ERROR');
    }
    this.baseUrl = options.baseUrl.replace(/\/+$/, '');
    const token = options.token;
    if (!options.anonymousOnly && (typeof token !== 'string' || token === '')) {
      throw new NekiroApiError(0, 'NeKiro development bearer token is required.', 'CONFIGURATION_ERROR');
    }
    if (typeof token === 'string' && (token !== token.trim() || /\s/.test(token))) {
      throw new Error('NeKiro bearer token must not contain whitespace');
    }
    this.token = token ?? '';
    this.publicAgentOrigin = options.publicAgentOrigin ?? '';
    this.fetchImpl = options.fetchImpl ?? ((input, init) => globalThis.fetch(input, init));
  }

  searchAgents(params: CatalogSearchParams = {}): Promise<CatalogSearchResponse> {
    const suffix = this.queryString(params);
    return this.request<unknown>('/v3/agents' + suffix).then((value) => validateCatalogSearchResponse(value, this.publicAgentOrigin));
  }

  resolvePublicAgent(publicAgentId: string): Promise<PublicAgentShare> {
    const safePublicAgentID = readPublicAgentID(publicAgentId);
    return this.publicRequest<unknown>('/v4/public/agents/' + encodeURIComponent(safePublicAgentID))
      .then((value) => validatePublicAgentShare(value, safePublicAgentID, this.publicAgentOrigin));
  }

  registerAgent(card: AgentCardV02): Promise<CatalogEntry> {
    return this.request<unknown>('/v3/agents', {
      method: 'POST',
      body: JSON.stringify({card}),
    }, 201).then((value) => validateCatalogEntry(value, this.publicAgentOrigin));
  }

  getAgentVersion(agentId: string, version: string): Promise<CatalogEntry> {
    return this.request<unknown>(this.versionPath(agentId, version)).then((value) => validateCatalogEntry(value, this.publicAgentOrigin));
  }

  publishAgentVersion(agentId: string, version: string): Promise<CatalogEntry> {
    return this.request<unknown>(this.versionPath(agentId, version) + '/publish', {method: 'POST'}).then((value) => validateCatalogEntry(value, this.publicAgentOrigin));
  }

  disableAgentVersion(agentId: string, version: string): Promise<CatalogEntry> {
    return this.request<unknown>(this.versionPath(agentId, version) + '/disable', {method: 'POST'}).then((value) => validateCatalogEntry(value, this.publicAgentOrigin));
  }

  createEndpointBinding(providerId: string, agentId: string, request: CreateEndpointBindingRequest): Promise<EndpointBinding> {
    const safeProviderId = readIdentifier(providerId, 'providerId');
    const safeAgentId = readIdentifier(agentId, 'agentId');
    const safeRequest = validateCreateEndpointBindingRequest(request);
    return this.trustedRequest<EndpointBinding>(
      '/v4/providers/' + encodeURIComponent(safeProviderId) + '/agents/' + encodeURIComponent(safeAgentId) + '/endpoint-bindings',
      {method: 'POST', body: JSON.stringify(safeRequest)},
      (value) => validateEndpointBinding(value, {providerId: safeProviderId, agentId: safeAgentId, version: safeRequest.version}),
      201,
    );
  }

  getEndpointBinding(providerId: string, bindingId: string): Promise<EndpointBinding> {
    const safeProviderId = readIdentifier(providerId, 'providerId');
    const safeBindingId = readIdentifier(bindingId, 'bindingId');
    return this.trustedRequest<EndpointBinding>(
      '/v4/providers/' + encodeURIComponent(safeProviderId) + '/endpoint-bindings/' + encodeURIComponent(safeBindingId),
      {},
      (value) => validateEndpointBinding(value, {providerId: safeProviderId, bindingId: safeBindingId}),
    );
  }

  createVerificationChallenge(providerId: string, bindingId: string): Promise<VerificationChallenge> {
    const safeProviderId = readIdentifier(providerId, 'providerId');
    const safeBindingId = readIdentifier(bindingId, 'bindingId');
    return this.trustedRequest<VerificationChallenge>(
      '/v4/providers/' + encodeURIComponent(safeProviderId) + '/endpoint-bindings/' + encodeURIComponent(safeBindingId) + '/challenges',
      {method: 'POST'},
      (value) => validateVerificationChallenge(value, safeBindingId),
      201,
    );
  }

  completeVerificationChallenge(providerId: string, bindingId: string, challengeId: string): Promise<EndpointBinding> {
    const safeProviderId = readIdentifier(providerId, 'providerId');
    const safeBindingId = readIdentifier(bindingId, 'bindingId');
    const safeChallengeId = readIdentifier(challengeId, 'challengeId');
    return this.trustedRequest<EndpointBinding>(
      '/v4/providers/' + encodeURIComponent(safeProviderId) + '/endpoint-bindings/' + encodeURIComponent(safeBindingId) + '/challenges/' + encodeURIComponent(safeChallengeId) + '/complete',
      {method: 'POST'},
      (value) => validateEndpointBinding(value, {providerId: safeProviderId, bindingId: safeBindingId}),
    );
  }

  createAgentRelease(providerId: string, agentId: string, request: CreateAgentReleaseRequest): Promise<AgentRelease> {
    const safeProviderId = readIdentifier(providerId, 'providerId');
    const safeAgentId = readIdentifier(agentId, 'agentId');
    const safeRequest = validateCreateAgentReleaseRequest(request);
    return this.trustedRequest<AgentRelease>(
      '/v4/providers/' + encodeURIComponent(safeProviderId) + '/agents/' + encodeURIComponent(safeAgentId) + '/releases',
      {method: 'POST', body: JSON.stringify(safeRequest)},
      (value) => validateAgentRelease(value, {
        providerId: safeProviderId,
        agentId: safeAgentId,
        version: safeRequest.version,
        bindingId: safeRequest.endpointBindingId,
      }),
      201,
    );
  }

  getAgentRelease(releaseId: string): Promise<AgentRelease> {
    const safeReleaseId = readIdentifier(releaseId, 'releaseId');
    return this.trustedRequest<AgentRelease>(
      '/v4/releases/' + encodeURIComponent(safeReleaseId),
      {},
      (value) => validateAgentRelease(value, {releaseId: safeReleaseId}),
    );
  }

  verifyAgentRelease(releaseId: string): Promise<AgentRelease> {
    return this.releaseAction(releaseId, 'verify');
  }

  publishAgentRelease(releaseId: string): Promise<AgentRelease> {
    return this.releaseAction(releaseId, 'publish');
  }

  suspendAgentRelease(releaseId: string): Promise<AgentRelease> {
    return this.releaseAction(releaseId, 'suspend');
  }

  revokeAgentRelease(releaseId: string): Promise<AgentRelease> {
    return this.releaseAction(releaseId, 'revoke');
  }

  createWorkspace(workspaceId: string): Promise<Workspace> {
    return this.request<unknown>('/v3/workspaces', {
      method: 'POST',
      body: JSON.stringify({workspaceId: readIdentifier(workspaceId, 'workspaceId')}),
    }, 201).then((value) => validateWorkspace(value));
  }

  getWorkspace(workspaceId: string): Promise<Workspace> {
    return this.request<unknown>('/v3/workspaces/' + encodeURIComponent(readIdentifier(workspaceId, 'workspaceId'))).then((value) => validateWorkspace(value));
  }

  installAgent(workspaceId: string, request: InstallAgentRequest): Promise<Installation> {
    return this.request<unknown>(this.workspaceInstallationPath(workspaceId), {
      method: 'POST',
      body: JSON.stringify({
        agentId: readIdentifier(request.agentId, 'agentId'),
        versionConstraint: readText(request.versionConstraint, 'versionConstraint'),
        acceptedPermissions: request.acceptedPermissions,
      }),
    }, 201).then((value) => validateInstallation(value, workspaceId));
  }

  listInstallations(workspaceId: string, params: {limit: number; cursor?: string}): Promise<InstallationList> {
    if (!params || !Number.isInteger(params.limit) || params.limit < 1 || params.limit > 100) {
      throw new Error('installation limit must be an integer between 1 and 100');
    }
    const query = this.queryString({limit: params.limit, cursor: params.cursor});
    return this.request<unknown>(this.workspaceInstallationPath(workspaceId) + query).then((value) => validateInstallationList(value, workspaceId));
  }

  getInstallation(workspaceId: string, installationId: string): Promise<Installation> {
    return this.request<unknown>(this.installationPath(workspaceId, installationId)).then((value) => validateInstallation(value, workspaceId));
  }

  async updateInstallation(workspaceId: string, installationId: string, status: Exclude<InstallationStatus, 'uninstalled'>): Promise<Installation> {
    const previous = await this.getInstallation(workspaceId, installationId);
    const value = await this.request<unknown>(this.installationPath(workspaceId, installationId), {
      method: 'PATCH',
      body: JSON.stringify({status}),
    }).then((response) => validateInstallation(response, workspaceId));
    return validateInstallationLifecycleResponse(value, previous);
  }

  async uninstallAgent(workspaceId: string, installationId: string): Promise<Installation> {
    const previous = await this.getInstallation(workspaceId, installationId);
    const value = await this.request<unknown>(this.installationPath(workspaceId, installationId), {method: 'DELETE'}).then((response) => validateInstallation(response, workspaceId));
    return validateInstallationLifecycleResponse(value, previous);
  }

  invoke(workspaceId: string, request: InvocationRequestV4): Promise<InvocationResultV1> {
    if (request.stream !== false) {
      throw new Error('streaming invocation must use invokeStream');
    }
    requireInvocationInput(request.input);
    return this.request<InvocationResultV1>(this.invocationPath(workspaceId), {
      method: 'POST',
      headers: {'Accept': 'application/json'},
      body: JSON.stringify({
        agentId: readIdentifier(request.agentId, 'agentId'),
        capability: readIdentifier(request.capability, 'capability'),
        input: request.input,
        stream: false,
      }),
    }).then((value) => validateInvocationResult(value));
  }

  async invokeStream(workspaceId: string, request: Omit<InvocationRequestV4, 'stream'>, onEvent?: (event: InvocationResultStreamEventV2) => void): Promise<InvocationResultStreamEventV2[]> {
    requireInvocationInput(request.input);
    const path = this.invocationPath(workspaceId);
    const response = await this.rawRequest(path, {
      method: 'POST',
      headers: {'Accept': 'text/event-stream'},
      body: JSON.stringify({
        agentId: readIdentifier(request.agentId, 'agentId'),
        capability: readIdentifier(request.capability, 'capability'),
        input: request.input,
        stream: true,
      }),
    });
    if (!response.ok) {
      throw await this.errorFromResponse(response);
    }
    if (response.headers.get('content-type')?.split(';', 1)[0].trim() !== 'text/event-stream') {
      throw new NekiroApiError(response.status, 'NeKiro Control Plane API returned an invalid stream media type.', 'INVALID_RESPONSE');
    }
    if (!response.body) {
      throw new NekiroApiError(response.status, 'NeKiro Control Plane API returned an empty stream.', 'INVALID_RESPONSE');
    }
    const reader = response.body.getReader();
    const decoder = new TextDecoder();
    let buffer = '';
    let dataLines: string[] = [];
    let expectedSequence = 0;
    let expectedChunkIndex = 0;
    let terminal = false;
    const events: InvocationResultStreamEventV2[] = [];
    const consume = (data: string) => {
      const event = validateResultStreamEvent(parseJsonValue(data, 'stream event'));
      if (terminal) throw new NekiroApiError(response.status, 'NeKiro stream emitted an event after terminal state.', 'INVALID_RESPONSE');
      if (event.sequence !== expectedSequence) throw new NekiroApiError(response.status, 'NeKiro stream sequence is not contiguous.', 'INVALID_RESPONSE');
      if ((expectedSequence === 0 && event.type !== 'accepted') || (expectedSequence > 0 && event.type === 'accepted')) throw new NekiroApiError(response.status, 'NeKiro stream accepted event must be first.', 'INVALID_RESPONSE');
      if (event.type === 'chunk') {
        if (event.chunkIndex !== expectedChunkIndex) throw new NekiroApiError(response.status, 'NeKiro stream chunk index is not contiguous.', 'INVALID_RESPONSE');
        expectedChunkIndex += 1;
      }
      if (events[0] && (event.invocationId !== events[0].invocationId || event.rootTaskId !== events[0].rootTaskId || event.traceId !== events[0].traceId)) {
        throw new NekiroApiError(response.status, 'NeKiro stream correlation changed.', 'INVALID_RESPONSE');
      }
      if (event.error && (event.error.invocationId !== event.invocationId || event.error.rootTaskId !== event.rootTaskId || event.error.traceId !== event.traceId)) {
        throw new NekiroApiError(response.status, 'NeKiro stream error correlation changed.', 'INVALID_RESPONSE');
      }
      expectedSequence += 1;
      terminal = event.type === 'completed' || event.type === 'failed' || event.type === 'canceled' || event.type === 'timed_out';
      events.push(event);
      onEvent?.(event);
    };
    const dispatchEvent = () => {
      if (dataLines.length === 0) return;
      const data = dataLines.join('\n');
      dataLines = [];
      consume(data);
    };
    const consumeLine = (line: string) => {
      if (line === '') {
        dispatchEvent();
        return;
      }
      if (line.startsWith(':')) return;
      const separator = line.indexOf(':');
      if (separator < 0) throw new NekiroApiError(response.status, 'NeKiro stream contains an invalid field.', 'INVALID_RESPONSE');
      const field = line.slice(0, separator);
      let value = line.slice(separator + 1);
      if (value.startsWith(' ')) value = value.slice(1);
      if (field === 'data') {
        dataLines.push(value);
      } else if (field !== 'event' && field !== 'id' && field !== 'retry') {
        throw new NekiroApiError(response.status, 'NeKiro stream contains an invalid field.', 'INVALID_RESPONSE');
      }
    };
    for (;;) {
      const result = await reader.read();
      buffer += decoder.decode(result.value ?? new Uint8Array(), {stream: !result.done});
      let newline = buffer.indexOf('\n');
      while (newline >= 0) {
        const line = buffer.slice(0, newline).replace(/\r$/, '');
        buffer = buffer.slice(newline + 1);
        consumeLine(line);
        newline = buffer.indexOf('\n');
      }
      if (result.done) break;
    }
    if (buffer !== '') consumeLine(buffer.replace(/\r$/, ''));
    dispatchEvent();
    if (!terminal) throw new NekiroApiError(response.status, 'NeKiro stream ended before a terminal event.', 'INVALID_RESPONSE');
    return events;
  }

  getInvocation(workspaceId: string, invocationId: string): Promise<InvocationDetailResponseV4> {
    return this.request<InvocationDetailResponseV4>(this.invocationPath(workspaceId) + '/' + encodeURIComponent(readIdentifier(invocationId, 'invocationId'))).then((value) => validateInvocationDetail(value, workspaceId));
  }

  getTrace(workspaceId: string, traceId: string): Promise<TraceResponseV4> {
    return this.request<TraceResponseV4>(this.tracePath(workspaceId, traceId)).then((value) => validateTrace(value, workspaceId, traceId));
  }

  private versionPath(agentId: string, version: string): string {
    return '/v3/agents/' + encodeURIComponent(agentId) + '/versions/' + encodeURIComponent(version);
  }

  private workspaceInstallationPath(workspaceId: string): string {
    return '/v3/workspaces/' + encodeURIComponent(readIdentifier(workspaceId, 'workspaceId')) + '/installations';
  }

  private installationPath(workspaceId: string, installationId: string): string {
    return this.workspaceInstallationPath(workspaceId) + '/' + encodeURIComponent(readIdentifier(installationId, 'installationId'));
  }

  private invocationPath(workspaceId: string): string {
    return '/v4/workspaces/' + encodeURIComponent(readIdentifier(workspaceId, 'workspaceId')) + '/invocations';
  }

  private tracePath(workspaceId: string, traceId: string): string {
    return '/v4/workspaces/' + encodeURIComponent(readIdentifier(workspaceId, 'workspaceId')) + '/traces/' + encodeURIComponent(readIdentifier(traceId, 'traceId'));
  }

  private releaseAction(releaseId: string, action: 'verify' | 'publish' | 'suspend' | 'revoke'): Promise<AgentRelease> {
    const safeReleaseId = readIdentifier(releaseId, 'releaseId');
    return this.trustedRequest<AgentRelease>(
      '/v4/releases/' + encodeURIComponent(safeReleaseId) + '/' + action,
      {method: 'POST'},
      (value) => validateAgentRelease(value, {releaseId: safeReleaseId}),
    );
  }

  private queryString(params: object): string {
    const query = new URLSearchParams();
    for (const [key, value] of Object.entries(params) as Array<[string, string | number | undefined]>) {
      if (value !== undefined && value !== '') {
        query.set(key, String(value));
      }
    }
    const serialized = query.toString();
    return serialized ? '?' + serialized : '';
  }

  private async request<T>(path: string, init: RequestInit = {}, expectedStatus = 200): Promise<T> {
    if (!this.baseUrl || !this.token) {
      throw new NekiroApiError(0, 'NeKiro Control Plane API base URL is not configured.', 'CONFIGURATION_ERROR');
    }

    const headers = new Headers(init.headers);
    headers.set('Accept', 'application/json');
    if (init.body && !headers.has('Content-Type')) {
      headers.set('Content-Type', 'application/json');
    }
    headers.set('Authorization', 'Bearer ' + this.token);

    const response = await this.rawRequest(path, {...init, headers});

    const responseText = await response.text();
    const payload = parseJson<unknown>(responseText);
    if (!response.ok) {
      throw await this.errorFromResponse(response, payload);
    }

    if (response.status !== expectedStatus) {
      throw new NekiroApiError(response.status, 'NeKiro Control Plane API returned an unexpected HTTP status.', 'INVALID_RESPONSE');
    }
    if (response.status === 204) {
      return undefined as T;
    }
    const mediaType = response.headers.get('content-type')?.split(';', 1)[0].trim();
    if (responseText.length === 0 || mediaType !== 'application/json' || payload === undefined) {
      throw new NekiroApiError(response.status, 'NeKiro Control Plane API returned an invalid JSON success response.', 'INVALID_RESPONSE');
    }
    return payload as T;
  }

  private async trustedRequest<T>(path: string, init: RequestInit, validate: (value: unknown) => T, expectedStatus = 200): Promise<T> {
    const response = await this.rawRequest(path, {...init, redirect: 'error'});
    const mediaType = response.headers.get('content-type')?.split(';', 1)[0].trim();
    const responseText = await response.text();
    const payload = parseJson<unknown>(responseText);
    if (!response.ok) {
      throw await this.trustedErrorFromResponse(response, payload);
    }
    if (response.status !== expectedStatus) {
      throw new NekiroApiError(response.status, 'NeKiro Trusted Publication returned an unexpected HTTP status.', 'INVALID_RESPONSE');
    }
    if (mediaType !== 'application/json' || payload === undefined) {
      throw new NekiroApiError(response.status, 'NeKiro Trusted Publication returned an invalid JSON response.', 'INVALID_RESPONSE');
    }
    try {
      return validate(payload);
    } catch {
      throw new NekiroApiError(response.status, 'NeKiro Trusted Publication returned an invalid response.', 'INVALID_RESPONSE');
    }
  }

  private async rawRequest(path: string, init: RequestInit = {}): Promise<Response> {
    if (!this.baseUrl || !this.token) throw new NekiroApiError(0, 'NeKiro authenticated API client is not configured.', 'CONFIGURATION_ERROR');
    const headers = new Headers(init.headers);
    headers.set('Accept', headers.get('Accept') ?? 'application/json');
    if (init.body && !headers.has('Content-Type')) headers.set('Content-Type', 'application/json');
    headers.set('Authorization', 'Bearer ' + this.token);
    try {
      return await this.fetchImpl(new URL(path, this.baseUrl + '/'), {...init, headers, redirect: 'error'});
    } catch {
      throw new NekiroApiError(0, 'NeKiro API request failed.', 'NETWORK_ERROR');
    }
  }

  private async publicRequest<T>(path: string): Promise<T> {
    if (!this.baseUrl) throw new NekiroApiError(0, 'NeKiro Control Plane API base URL is not configured.', 'CONFIGURATION_ERROR');
    const headers = new Headers({'Accept': 'application/json'});
    let response: Response;
    try {
      response = await this.fetchImpl(new URL(path, this.baseUrl + '/'), {headers, redirect: 'error'});
    } catch {
      throw new NekiroApiError(0, 'NeKiro API request failed.', 'NETWORK_ERROR');
    }
    const responseText = await response.text();
    const payload = parseJson<unknown>(responseText);
    if (!response.ok) throw await this.errorFromResponse(response, payload);
    if (response.status !== 200 || response.headers.get('content-type')?.split(';', 1)[0].trim() !== 'application/json' || payload === undefined) {
      throw new NekiroApiError(response.status, 'NeKiro public Agent response is invalid.', 'INVALID_RESPONSE');
    }
    return payload as T;
  }

  private async errorFromResponse(response: Response, knownPayload?: unknown): Promise<NekiroApiError> {
    if (response.headers.get('content-type')?.split(';', 1)[0].trim() !== 'application/json') {
      return new NekiroApiError(response.status, 'NeKiro Control Plane API returned an invalid Platform Error payload.', 'INVALID_RESPONSE');
    }
    const payload = knownPayload ?? parseJson<unknown>(await response.text());
    if (!isPlatformErrorV4(payload)) {
      return new NekiroApiError(response.status, 'NeKiro Control Plane API returned an invalid Platform Error payload.', 'INVALID_RESPONSE');
    }
    const headerTraceId = response.headers.get('x-nek-trace-id');
    if (headerTraceId !== null && (headerTraceId !== payload.traceId || !/^[A-Za-z0-9](?:[A-Za-z0-9._:-]{0,127})$/.test(headerTraceId))) {
      return new NekiroApiError(response.status, 'NeKiro Control Plane API returned inconsistent trace correlation.', 'INVALID_RESPONSE');
    }
    const invocationId = 'invocationId' in payload ? payload.invocationId : undefined;
    const rootTaskId = 'rootTaskId' in payload ? payload.rootTaskId : undefined;
    return new NekiroApiError(response.status, payload.message, payload.code, payload.traceId, invocationId, rootTaskId);
  }

  private trustedErrorFromResponse(response: Response, payload: unknown): NekiroApiError {
    if (response.headers.get('content-type')?.split(';', 1)[0].trim() !== 'application/json') {
      return new NekiroApiError(response.status, 'NeKiro Trusted Publication returned an invalid error response.', 'INVALID_RESPONSE');
    }
    if (!isTrustedPublicationError(payload)) {
      return new NekiroApiError(response.status, 'NeKiro Trusted Publication returned an invalid error response.', 'INVALID_RESPONSE');
    }
    const headerTraceId = response.headers.get('x-nek-trace-id');
    if (headerTraceId !== null && (headerTraceId !== payload.traceId || !/^[A-Za-z0-9](?:[A-Za-z0-9._:-]{0,127})$/.test(headerTraceId))) {
      return new NekiroApiError(response.status, 'NeKiro Trusted Publication returned inconsistent trace correlation.', 'INVALID_RESPONSE');
    }
    return new NekiroApiError(response.status, payload.message, payload.code, payload.traceId);
  }
}

const PLATFORM_ERROR_MESSAGES: Record<PlatformErrorCode, string> = {
  VALIDATION_ERROR: 'The request is invalid.',
  UNAUTHENTICATED: 'Authentication is required.',
  FORBIDDEN: 'The requested operation is not allowed.',
  NOT_FOUND: 'The requested resource was not found.',
  CONFLICT: 'The requested operation conflicts with current state.',
  NOT_ACCEPTABLE: 'The requested result mode is not acceptable.',
  PAYLOAD_TOO_LARGE: 'The payload is too large.',
  AGENT_NOT_INSTALLED: 'The Agent is not installed in this Workspace.',
  INSTALLATION_DISABLED: 'The Agent installation is disabled.',
  AGENT_DISABLED: 'The Agent version is disabled.',
  AGENT_RELEASE_UNPUBLISHED: 'The Agent release is not published.',
  AGENT_RELEASE_SUSPENDED: 'The Agent release is suspended.',
  AGENT_RELEASE_REVOKED: 'The Agent release is revoked.',
  CAPABILITY_NOT_ALLOWED: 'The requested capability is not allowed.',
  ROUTE_NOT_FOUND: 'No route is available for the Agent.',
  AGENT_AUTH_UNSUPPORTED: 'The Agent authentication type is not supported for invocation.',
  AGENT_RESPONSE_TOO_LARGE: 'The Agent response is too large.',
  A2A_PROTOCOL_ERROR: 'The Agent returned an invalid A2A response.',
  AGENT_UNAVAILABLE: 'The Agent is unavailable.',
  AGENT_EXECUTION_FAILED: 'The Agent failed to complete the invocation.',
  DEPENDENCY_ERROR: 'A required platform dependency failed.',
  TIMEOUT: 'The invocation timed out.',
  CANCELED: 'The invocation was canceled.',
  INTERNAL_ERROR: 'The platform could not complete the request.',
};

const TRUSTED_PUBLICATION_ERROR_CODES: readonly TrustedPublicationErrorCode[] = [
  'VALIDATION_ERROR', 'UNAUTHENTICATED', 'FORBIDDEN', 'NOT_FOUND', 'CONFLICT',
  'INVALID_ENDPOINT', 'DISALLOWED_NETWORK', 'ENDPOINT_UNAVAILABLE', 'WRONG_PROOF',
  'CHALLENGE_EXPIRED', 'CHALLENGE_REUSED', 'REDIRECT_NOT_ALLOWED',
  'DEPENDENCY_ERROR', 'INTERNAL_ERROR',
];

interface TrustedPublicationError {
  code: TrustedPublicationErrorCode;
  message: string;
  traceId: string;
}

function validateCreateEndpointBindingRequest(value: unknown): CreateEndpointBindingRequest {
  const record = requireRecord(value, 'Create Endpoint Binding request');
  assertAllowedKeys(record, ['endpoint', 'method', 'version'], 'Create Endpoint Binding request');
  const endpoint = requireHttpUri(record.endpoint, 'endpoint');
  const method = requireEnum(record.method, ['http_well_known'], 'method') as 'http_well_known';
  const version = requireSemver(record.version, 'version');
  return {endpoint, method, version};
}

function validateCreateAgentReleaseRequest(value: unknown): CreateAgentReleaseRequest {
  const record = requireRecord(value, 'Create Agent Release request');
  assertAllowedKeys(record, ['version', 'endpointBindingId'], 'Create Agent Release request');
  return {
    version: requireSemver(record.version, 'version'),
    endpointBindingId: readIdentifier(record.endpointBindingId, 'endpointBindingId'),
  };
}

function validateEndpointBinding(value: unknown, expected: {providerId?: string; agentId?: string; version?: string; bindingId?: string}): EndpointBinding {
  const record = requireRecord(value, 'Endpoint Binding');
  assertAllowedKeys(record, ['bindingId', 'providerId', 'agentId', 'agentCardVersion', 'endpoint', 'verificationMethod', 'verificationStatus', 'verificationFailureCode', 'verificationEvidenceDigest', 'createdAt', 'updatedAt', 'verifiedAt', 'revokedAt'], 'Endpoint Binding');
  const result: EndpointBinding = {
    bindingId: readIdentifier(record.bindingId, 'bindingId'),
    providerId: readIdentifier(record.providerId, 'providerId'),
    agentId: readIdentifier(record.agentId, 'agentId'),
    agentCardVersion: requireSemver(record.agentCardVersion, 'agentCardVersion'),
    endpoint: requireHttpUri(record.endpoint, 'endpoint'),
    verificationMethod: requireEnum(record.verificationMethod, ['http_well_known'], 'verificationMethod') as 'http_well_known',
    verificationStatus: requireEnum(record.verificationStatus, ['pending', 'verified', 'failed', 'revoked'], 'verificationStatus') as EndpointBindingVerificationStatus,
    createdAt: requireDateValue(record.createdAt, 'createdAt'),
    updatedAt: requireDateValue(record.updatedAt, 'updatedAt'),
  };
  if ('verificationFailureCode' in record) result.verificationFailureCode = requireOptionalText(record.verificationFailureCode, 'verificationFailureCode', 64);
  if ('verificationEvidenceDigest' in record) result.verificationEvidenceDigest = requireDigest(record.verificationEvidenceDigest, 'verificationEvidenceDigest');
  if ('verifiedAt' in record) result.verifiedAt = requireDateValue(record.verifiedAt, 'verifiedAt');
  if ('revokedAt' in record) result.revokedAt = requireDateValue(record.revokedAt, 'revokedAt');
  switch (result.verificationStatus) {
    case 'pending':
      rejectPresent(record, ['verificationFailureCode', 'verificationEvidenceDigest', 'verifiedAt', 'revokedAt'], 'pending Endpoint Binding');
      break;
    case 'verified':
      result.verificationEvidenceDigest = requireDigest(record.verificationEvidenceDigest, 'verificationEvidenceDigest');
      result.verifiedAt = requireDateValue(record.verifiedAt, 'verifiedAt');
      rejectPresent(record, ['verificationFailureCode', 'revokedAt'], 'verified Endpoint Binding');
      break;
    case 'failed':
      result.verificationFailureCode = requireBoundedText(record.verificationFailureCode, 'verificationFailureCode', 1, 64);
      rejectPresent(record, ['verificationEvidenceDigest', 'verifiedAt', 'revokedAt'], 'failed Endpoint Binding');
      break;
    case 'revoked':
      result.revokedAt = requireDateValue(record.revokedAt, 'revokedAt');
      rejectPresent(record, ['verificationFailureCode', 'verificationEvidenceDigest', 'verifiedAt'], 'revoked Endpoint Binding');
      break;
  }
  if (expected.providerId !== undefined && result.providerId !== expected.providerId) throw new Error('Endpoint Binding provider does not match the request');
  if (expected.agentId !== undefined && result.agentId !== expected.agentId) throw new Error('Endpoint Binding Agent does not match the request');
  if (expected.version !== undefined && result.agentCardVersion !== expected.version) throw new Error('Endpoint Binding version does not match the request');
  if (expected.bindingId !== undefined && result.bindingId !== expected.bindingId) throw new Error('Endpoint Binding ID does not match the request');
  return result;
}

function validateVerificationChallenge(value: unknown, expectedBindingId: string): VerificationChallenge {
  const record = requireRecord(value, 'Verification Challenge');
  assertAllowedKeys(record, ['challengeId', 'bindingId', 'challengeUrl', 'proof', 'expiresAt'], 'Verification Challenge');
  const result: VerificationChallenge = {
    challengeId: readIdentifier(record.challengeId, 'challengeId'),
    bindingId: readIdentifier(record.bindingId, 'bindingId'),
    challengeUrl: requireUri(record.challengeUrl, 'challengeUrl'),
    proof: requireBoundedText(record.proof, 'proof', 1, 128),
    expiresAt: requireDateValue(record.expiresAt, 'expiresAt'),
  };
  if (result.bindingId !== expectedBindingId) throw new Error('Verification Challenge Binding does not match the request');
  return result;
}

function validateAgentRelease(value: unknown, expected: {providerId?: string; agentId?: string; version?: string; bindingId?: string; releaseId?: string}): AgentRelease {
  const record = requireRecord(value, 'Agent Release');
  assertAllowedKeys(record, ['releaseId', 'providerId', 'agentId', 'agentCardVersion', 'cardDigest', 'endpointBindingId', 'endpointOrigin', 'endpointPath', 'verificationMethod', 'verificationEvidenceDigest', 'state', 'createdAt', 'updatedAt', 'verifiedAt', 'publishedAt', 'suspendedAt', 'revokedAt'], 'Agent Release');
  const result: AgentRelease = {
    releaseId: readIdentifier(record.releaseId, 'releaseId'),
    providerId: readIdentifier(record.providerId, 'providerId'),
    agentId: readIdentifier(record.agentId, 'agentId'),
    agentCardVersion: requireSemver(record.agentCardVersion, 'agentCardVersion'),
    cardDigest: requireDigest(record.cardDigest, 'cardDigest'),
    endpointBindingId: readIdentifier(record.endpointBindingId, 'endpointBindingId'),
    endpointOrigin: requireHttpUri(record.endpointOrigin, 'endpointOrigin'),
    endpointPath: requireBoundedText(record.endpointPath, 'endpointPath', 1),
    verificationMethod: requireEnum(record.verificationMethod, ['http_well_known'], 'verificationMethod') as 'http_well_known',
    state: requireEnum(record.state, ['draft', 'pending_verification', 'verified', 'published', 'suspended', 'revoked'], 'state') as AgentReleaseState,
    createdAt: requireDateValue(record.createdAt, 'createdAt'),
    updatedAt: requireDateValue(record.updatedAt, 'updatedAt'),
  };
  if ('verificationEvidenceDigest' in record) result.verificationEvidenceDigest = requireDigest(record.verificationEvidenceDigest, 'verificationEvidenceDigest');
  if ('verifiedAt' in record) result.verifiedAt = requireDateValue(record.verifiedAt, 'verifiedAt');
  if ('publishedAt' in record) result.publishedAt = requireDateValue(record.publishedAt, 'publishedAt');
  if ('suspendedAt' in record) result.suspendedAt = requireDateValue(record.suspendedAt, 'suspendedAt');
  if ('revokedAt' in record) result.revokedAt = requireDateValue(record.revokedAt, 'revokedAt');
  switch (result.state) {
    case 'draft':
    case 'pending_verification':
      rejectPresent(record, ['verificationEvidenceDigest', 'verifiedAt', 'publishedAt', 'suspendedAt', 'revokedAt'], result.state + ' Agent Release');
      break;
    case 'verified':
      result.verificationEvidenceDigest = requireDigest(record.verificationEvidenceDigest, 'verificationEvidenceDigest');
      result.verifiedAt = requireDateValue(record.verifiedAt, 'verifiedAt');
      rejectPresent(record, ['publishedAt', 'suspendedAt', 'revokedAt'], 'verified Agent Release');
      break;
    case 'published':
      result.verificationEvidenceDigest = requireDigest(record.verificationEvidenceDigest, 'verificationEvidenceDigest');
      result.verifiedAt = requireDateValue(record.verifiedAt, 'verifiedAt');
      result.publishedAt = requireDateValue(record.publishedAt, 'publishedAt');
      rejectPresent(record, ['suspendedAt', 'revokedAt'], 'published Agent Release');
      break;
    case 'suspended':
      result.verificationEvidenceDigest = requireDigest(record.verificationEvidenceDigest, 'verificationEvidenceDigest');
      result.verifiedAt = requireDateValue(record.verifiedAt, 'verifiedAt');
      result.suspendedAt = requireDateValue(record.suspendedAt, 'suspendedAt');
      rejectPresent(record, ['revokedAt'], 'suspended Agent Release');
      break;
    case 'revoked':
      result.verificationEvidenceDigest = requireDigest(record.verificationEvidenceDigest, 'verificationEvidenceDigest');
      result.verifiedAt = requireDateValue(record.verifiedAt, 'verifiedAt');
      result.revokedAt = requireDateValue(record.revokedAt, 'revokedAt');
      break;
  }
  if (expected.providerId !== undefined && result.providerId !== expected.providerId) throw new Error('Agent Release provider does not match the request');
  if (expected.agentId !== undefined && result.agentId !== expected.agentId) throw new Error('Agent Release Agent does not match the request');
  if (expected.version !== undefined && result.agentCardVersion !== expected.version) throw new Error('Agent Release version does not match the request');
  if (expected.bindingId !== undefined && result.endpointBindingId !== expected.bindingId) throw new Error('Agent Release Binding does not match the request');
  if (expected.releaseId !== undefined && result.releaseId !== expected.releaseId) throw new Error('Agent Release ID does not match the request');
  return result;
}

function isTrustedPublicationError(value: unknown): value is TrustedPublicationError {
  if (!isRecord(value)) return false;
  if (Object.keys(value).some((key) => !['code', 'message', 'traceId'].includes(key))) return false;
  return typeof value.code === 'string'
    && TRUSTED_PUBLICATION_ERROR_CODES.includes(value.code as TrustedPublicationErrorCode)
    && typeof value.message === 'string'
    && value.message.length > 0
    && typeof value.traceId === 'string'
    && /^[A-Za-z0-9](?:[A-Za-z0-9._:-]{0,127})$/.test(value.traceId);
}

function rejectPresent(record: Record<string, unknown>, fields: string[], state: string): void {
  for (const field of fields) {
    if (field in record) throw new Error(`${field} must be absent for ${state}`);
  }
}

function validateInvocationResult(value: unknown): InvocationResultV1 {
  const record = requireRecord(value, 'Invocation Result');
  assertAllowedKeys(record, ['schemaVersion', 'invocationId', 'rootTaskId', 'traceId', 'status', 'result'], 'Invocation Result');
  if (record.schemaVersion !== '1' || record.status !== 'succeeded') throw new Error('Invocation Result schema or status is invalid');
  requireIdentifier(record.invocationId, 'invocationId');
  requireIdentifier(record.rootTaskId, 'rootTaskId');
  requireIdentifier(record.traceId, 'traceId');
  if (!('result' in record)) throw new Error('Invocation Result result is required');
  return record as unknown as InvocationResultV1;
}

function validateResultStreamEvent(value: unknown): InvocationResultStreamEventV2 {
  const record = requireRecord(value, 'Invocation Result Stream Event');
  assertAllowedKeys(record, ['schemaVersion', 'sequence', 'type', 'status', 'invocationId', 'rootTaskId', 'traceId', 'chunkIndex', 'chunk', 'error'], 'Invocation Result Stream Event');
  if (record.schemaVersion !== '2' || typeof record.sequence !== 'number' || !Number.isInteger(record.sequence) || record.sequence < 0) throw new Error('Invocation Result Stream Event schema or sequence is invalid');
  const type = requireEnum(record.type, ['accepted', 'chunk', 'completed', 'failed', 'canceled', 'timed_out'], 'stream event type') as ResultStreamEventType;
  const status = requireEnum(record.status, ['pending', 'running', 'succeeded', 'failed', 'canceled', 'timed_out'], 'stream event status') as InvocationResultStatus;
  requireIdentifier(record.invocationId, 'invocationId'); requireIdentifier(record.rootTaskId, 'rootTaskId'); requireIdentifier(record.traceId, 'traceId');
  if (type === 'accepted' && (status !== 'pending' || 'chunk' in record || 'chunkIndex' in record || 'error' in record)) throw new Error('accepted stream event is invalid');
  if (type === 'chunk' && (status !== 'running' || !isNonNegativeInteger(record.chunkIndex) || !('chunk' in record) || 'error' in record)) throw new Error('chunk stream event is invalid');
  if (type === 'completed' && (status !== 'succeeded' || 'chunk' in record || 'chunkIndex' in record || 'error' in record)) throw new Error('completed stream event is invalid');
  if (type === 'failed' || type === 'canceled' || type === 'timed_out') {
    if (status !== type) throw new Error('terminal stream event status is invalid');
    if (!isCorrelatedPlatformError(record.error)) throw new Error('terminal stream event error is invalid');
    if (type === 'failed' && (record.error.code === 'CANCELED' || record.error.code === 'TIMEOUT')) throw new Error('failed stream event error code is invalid');
    if (type === 'canceled' && record.error.code !== 'CANCELED') throw new Error('canceled stream event error code is invalid');
    if (type === 'timed_out' && record.error.code !== 'TIMEOUT') throw new Error('timed_out stream event error code is invalid');
    if ('chunk' in record || 'chunkIndex' in record) throw new Error('terminal stream event cannot contain a chunk');
  }
  return record as unknown as InvocationResultStreamEventV2;
}

function validateInvocationDetail(value: unknown, workspaceId: string): InvocationDetailResponseV4 {
  const record = requireRecord(value, 'Invocation Detail');
  assertAllowedKeys(record, ['invocation', 'events'], 'Invocation Detail');
  const invocation = validateInvocationRecord(record.invocation, workspaceId);
  if (!Array.isArray(record.events) || record.events.length === 0) throw new Error('Invocation Detail events are required');
  const events = record.events.map((event) => validateInvocationEvent(event, workspaceId));
  const lastEvent = events[events.length - 1];
  const eventIDs = new Set<string>();
  let previous: InvocationEventV03 | undefined;
  let expectedChunkIndex = 0;
  events.forEach((event, index) => {
    if (event.sequence !== index || event.invocationId !== invocation.invocationId || event.rootTaskId !== invocation.rootTaskId || event.parentInvocationId !== invocation.parentInvocationId || event.traceId !== invocation.traceId || event.workspaceId !== invocation.workspaceId || event.targetAgentId !== invocation.targetAgentId || event.agentCardVersion !== invocation.agentCardVersion || event.agentReleaseId !== invocation.agentReleaseId || event.agentCardDigest !== invocation.agentCardDigest || event.capability !== invocation.capability || event.caller.type !== invocation.caller.type || event.caller.id !== invocation.caller.id) throw new Error('Invocation Detail event correlation is invalid');
    if (eventIDs.has(event.eventId)) throw new Error('Invocation Detail repeats an event');
    eventIDs.add(event.eventId);
    if (!previous) {
      if (event.type !== 'created' || event.status !== 'pending') throw new Error('Invocation Detail must begin with created/pending');
    } else if (!validInvocationTransition(previous.status, event.type, event.status)) {
      throw new Error('Invocation Detail event transition is invalid');
    }
    if (event.type === 'stream') {
      if (event.chunkIndex !== expectedChunkIndex) throw new Error('Invocation Detail chunk sequence is invalid');
      expectedChunkIndex += 1;
    }
    previous = event;
  });
  if (lastEvent.status !== invocation.status) throw new Error('Invocation Detail status does not match its last event');
  return {invocation, events};
}

function validateTrace(value: unknown, workspaceId: string, traceId: string): TraceResponseV4 {
  const record = requireRecord(value, 'Trace');
  assertAllowedKeys(record, ['traceId', 'invocations'], 'Trace');
  if (record.traceId !== traceId || !Array.isArray(record.invocations) || record.invocations.length === 0) throw new Error('Trace correlation or lineage is invalid');
  const invocations = record.invocations.map((item) => validateInvocationRecord(item, workspaceId));
  const identities = new Set<string>();
  const rootTaskID = invocations[0].rootTaskId;
  invocations.forEach((invocation) => {
    if (invocation.traceId !== traceId || invocation.rootTaskId !== rootTaskID || identities.has(invocation.invocationId)) throw new Error('Trace Invocation correlation is invalid');
    if (invocation.parentInvocationId && (invocation.parentInvocationId === invocation.invocationId || !identities.has(invocation.parentInvocationId))) throw new Error('Trace parent ordering is invalid');
    identities.add(invocation.invocationId);
  });
  return {traceId, invocations};
}

function validateInvocationRecord(value: unknown, workspaceId: string): InvocationRecordV4 {
  const record = requireRecord(value, 'Invocation Record');
  assertAllowedKeys(record, ['invocationId', 'rootTaskId', 'parentInvocationId', 'traceId', 'caller', 'workspaceId', 'targetAgentId', 'agentCardVersion', 'agentReleaseId', 'agentCardDigest', 'capability', 'status', 'latencyMs', 'errorCode', 'createdAt', 'updatedAt'], 'Invocation Record');
  const invocationId = record.invocationId;
  requireIdentifier(invocationId, 'invocationId');
  requireIdentifier(record.rootTaskId, 'rootTaskId');
  requireIdentifier(record.traceId, 'traceId');
  if (record.workspaceId !== workspaceId) throw new Error('Invocation Record Workspace does not match the active Workspace');
  requireIdentifier(record.targetAgentId, 'targetAgentId'); requireIdentifier(record.capability, 'capability');
  if (typeof record.agentCardVersion !== 'string' || !isSemver(record.agentCardVersion)) throw new Error('agentCardVersion must be strict SemVer');
  validateReleaseProvenance(record.agentReleaseId, record.agentCardDigest, 'Invocation Record');
  requireCaller(record.caller);
  requireEnum(record.status, ['pending', 'routing', 'running', 'succeeded', 'failed', 'canceled', 'timed_out'], 'invocation status');
  requireDate(record.createdAt, 'createdAt'); requireDate(record.updatedAt, 'updatedAt');
  if ('parentInvocationId' in record && record.parentInvocationId !== undefined) requireIdentifier(record.parentInvocationId, 'parentInvocationId');
  if ('latencyMs' in record && record.latencyMs !== undefined && (typeof record.latencyMs !== 'number' || !Number.isInteger(record.latencyMs) || record.latencyMs < 0)) throw new Error('latencyMs is invalid');
  if ('errorCode' in record && record.errorCode !== undefined) requirePlatformCode(record.errorCode);
  return {...record, invocationId} as unknown as InvocationRecordV4;
}

function validateInvocationEvent(value: unknown, workspaceId: string): InvocationEventV03 {
  const record = requireRecord(value, 'Invocation Event');
  assertAllowedKeys(record, ['schemaVersion', 'eventId', 'sequence', 'occurredAt', 'type', 'status', 'invocationId', 'rootTaskId', 'parentInvocationId', 'traceId', 'caller', 'workspaceId', 'targetAgentId', 'agentCardVersion', 'agentReleaseId', 'agentCardDigest', 'capability', 'chunkIndex', 'chunkBytes', 'latencyMs', 'error'], 'Invocation Event');
  if (record.schemaVersion !== '0.3' || typeof record.sequence !== 'number' || !Number.isInteger(record.sequence) || record.sequence < 0) throw new Error('Invocation Event schema or sequence is invalid');
  requireIdentifier(record.eventId, 'eventId'); requireDate(record.occurredAt, 'occurredAt'); requireIdentifier(record.invocationId, 'invocationId'); requireIdentifier(record.rootTaskId, 'rootTaskId'); requireIdentifier(record.traceId, 'traceId');
  if (record.workspaceId !== workspaceId) throw new Error('Invocation Event Workspace does not match the active Workspace');
  requireCaller(record.caller); requireIdentifier(record.targetAgentId, 'targetAgentId');
  if (typeof record.agentCardVersion !== 'string' || !isSemver(record.agentCardVersion)) throw new Error('agentCardVersion must be strict SemVer');
  validateReleaseProvenance(record.agentReleaseId, record.agentCardDigest, 'Invocation Event');
  requireIdentifier(record.capability, 'capability');
  const type = requireEnum(record.type, ['created', 'routing', 'started', 'stream', 'succeeded', 'failed', 'canceled', 'timed_out'], 'event type') as InvocationEventType;
  const status = requireEnum(record.status, ['pending', 'routing', 'running', 'succeeded', 'failed', 'canceled', 'timed_out'], 'event status') as InvocationEventStatus;
  if (type === 'created' && (status !== 'pending' || hasAny(record, ['chunkIndex', 'chunkBytes', 'latencyMs', 'error']))) throw new Error('created Invocation Event is invalid');
  if (type === 'routing' && (status !== 'routing' || hasAny(record, ['chunkIndex', 'chunkBytes', 'latencyMs', 'error']))) throw new Error('routing Invocation Event is invalid');
  if (type === 'started' && (status !== 'running' || hasAny(record, ['chunkIndex', 'chunkBytes', 'latencyMs', 'error']))) throw new Error('started Invocation Event is invalid');
  if (type === 'stream' && (status !== 'running' || !isNonNegativeInteger(record.chunkIndex) || !isNonNegativeInteger(record.chunkBytes) || hasAny(record, ['latencyMs', 'error']))) throw new Error('stream Invocation Event is invalid');
  if (type === 'succeeded' && (status !== 'succeeded' || !isNonNegativeInteger(record.latencyMs) || hasAny(record, ['chunkIndex', 'chunkBytes', 'error']))) throw new Error('succeeded Invocation Event is invalid');
  if (type === 'failed' || type === 'canceled' || type === 'timed_out') {
    if (status !== type || !isNonNegativeInteger(record.latencyMs) || !isCorrelatedPlatformError(record.error) || hasAny(record, ['chunkIndex', 'chunkBytes'])) throw new Error('terminal Invocation Event is invalid');
    if (type === 'failed' && (record.error.code === 'CANCELED' || record.error.code === 'TIMEOUT')) throw new Error('failed Invocation Event error code is invalid');
    if (type === 'canceled' && record.error.code !== 'CANCELED') throw new Error('canceled Invocation Event error code is invalid');
    if (type === 'timed_out' && record.error.code !== 'TIMEOUT') throw new Error('timed_out Invocation Event error code is invalid');
  }
  if (('chunkIndex' in record && record.chunkIndex !== undefined && !isNonNegativeInteger(record.chunkIndex)) || ('chunkBytes' in record && record.chunkBytes !== undefined && !isNonNegativeInteger(record.chunkBytes))) throw new Error('Invocation Event chunk metadata is invalid');
  if (isCorrelatedPlatformError(record.error) && (record.error.invocationId !== record.invocationId || record.error.rootTaskId !== record.rootTaskId || record.error.traceId !== record.traceId)) throw new Error('Invocation Event error correlation changed');
  return record as unknown as InvocationEventV03;
}

function isCorrelatedPlatformError(value: unknown): value is CorrelatedPlatformErrorV4 {
  if (!isRecord(value) || !('invocationId' in value) || !('rootTaskId' in value)) return false;
  if (Object.keys(value).some((key) => !['code', 'message', 'traceId', 'invocationId', 'rootTaskId'].includes(key))) return false;
  requirePlatformCode(value.code);
  if (value.message !== PLATFORM_ERROR_MESSAGES[value.code as PlatformErrorCode]) return false;
  requireIdentifier(value.traceId, 'traceId'); requireIdentifier(value.invocationId, 'invocationId'); requireIdentifier(value.rootTaskId, 'rootTaskId');
  return true;
}

function requirePlatformCode(value: unknown): asserts value is PlatformErrorCode {
  requireEnum(value, Object.keys(PLATFORM_ERROR_MESSAGES), 'Platform Error code');
}

function requireCaller(value: unknown): void {
  const caller = requireRecord(value, 'caller');
  requireEnum(caller.type, ['user', 'agent', 'service'], 'caller.type'); requireIdentifier(caller.id, 'caller.id');
}

function requireSemver(value: unknown, field: string): string {
  if (typeof value !== 'string' || !isSemver(value)) throw new Error(field + ' must be strict SemVer');
  return value;
}

function requireUri(value: unknown, field: string): string {
  if (typeof value !== 'string' || value.length === 0) throw new Error(field + ' must be a URI');
  if (value !== value.trim() || /\s/.test(value)) throw new Error(field + ' must not contain whitespace');
  try {
    const parsed = new URL(value);
    if (parsed.username || parsed.password || hasUriUserinfo(value)) throw new Error('URI userinfo is not allowed');
  } catch {
    throw new Error(field + ' must be a valid URI without userinfo');
  }
  return value;
}

function requireHttpUri(value: unknown, field: string): string {
  const uri = requireUri(value, field);
  const parsed = new URL(uri);
  if (parsed.protocol !== 'http:' && parsed.protocol !== 'https:') throw new Error(field + ' must be an HTTP(S) URI');
  return uri;
}

function requireDateValue(value: unknown, field: string): string {
  requireDate(value, field);
  return value as string;
}

function requireDigest(value: unknown, field: string): string {
  if (typeof value !== 'string' || !/^[0-9a-f]{64}$/.test(value)) throw new Error(field + ' must be a lowercase 64-hex digest');
  return value;
}

function requireBoundedText(value: unknown, field: string, minimum: number, maximum?: number): string {
  if (typeof value !== 'string' || value.length < minimum || (maximum !== undefined && value.length > maximum)) {
    throw new Error(field + ' has an invalid length');
  }
  return value;
}

function requireOptionalText(value: unknown, field: string, maximum: number): string {
  return requireBoundedText(value, field, 0, maximum);
}

function requireDate(value: unknown, field: string): void {
  if (typeof value !== 'string' || !isStrictDateTime(value)) throw new Error(field + ' is invalid');
}

function isStrictDateTime(value: string): boolean {
  const match = /^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2}):(\d{2})(?:\.\d+)?(Z|[+-]\d{2}:\d{2})$/.exec(value);
  if (!match) return false;
  const year = Number(match[1]);
  const month = Number(match[2]);
  const day = Number(match[3]);
  const hour = Number(match[4]);
  const minute = Number(match[5]);
  const second = Number(match[6]);
  if (month < 1 || month > 12 || day < 1 || day > daysInMonth(year, month) || hour > 23 || minute > 59 || second > 59) return false;
  if (match[7] !== 'Z') {
    const offsetHours = Number(match[7].slice(1, 3));
    const offsetMinutes = Number(match[7].slice(4, 6));
    if (offsetHours > 23 || offsetMinutes > 59) return false;
  }
  return true;
}

function daysInMonth(year: number, month: number): number {
  if (month === 2) return year % 4 === 0 && (year % 100 !== 0 || year % 400 === 0) ? 29 : 28;
  return [4, 6, 9, 11].includes(month) ? 30 : 31;
}

function requireIdentifier(value: unknown, field: string): asserts value is string {
  if (typeof value !== 'string' || !/^[A-Za-z0-9](?:[A-Za-z0-9._:-]{0,127})$/.test(value)) throw new Error(field + ' must be a NeKiro safe identifier');
}

function requireEnum(value: unknown, values: readonly string[], field: string): string {
  if (typeof value !== 'string' || !values.includes(value)) throw new Error(field + ' is invalid');
  return value;
}

function isNonNegativeInteger(value: unknown): value is number {
  return typeof value === 'number' && Number.isInteger(value) && value >= 0;
}

function hasAny(record: Record<string, unknown>, fields: string[]): boolean {
  return fields.some((field) => field in record);
}

function assertAllowedKeys(record: Record<string, unknown>, allowed: string[], field: string): void {
  const known = new Set(allowed);
  if (Object.keys(record).some((key) => !known.has(key))) throw new Error(field + ' contains an unknown field');
}

function validInvocationTransition(from: InvocationEventStatus, type: InvocationEventType, to: InvocationEventStatus): boolean {
  if (from === 'pending') return (type === 'routing' && to === 'routing') || ((type === 'canceled' || type === 'timed_out') && type === to);
  if (from === 'routing') return (type === 'started' && to === 'running') || ((type === 'failed' || type === 'canceled' || type === 'timed_out') && type === to);
  if (from === 'running') return (type === 'stream' && to === 'running') || ((type === 'succeeded' || type === 'failed' || type === 'canceled' || type === 'timed_out') && type === to);
  return false;
}

function requireRecord(value: unknown, field: string): Record<string, unknown> {
  if (!isRecord(value)) throw new Error(field + ' must be a JSON object');
  return value;
}

function requireInvocationInput(value: unknown): asserts value is JsonObject {
  if (!isRecord(value)) throw new Error('invocation input must be a JSON object');
}

function parseJsonValue(value: string, field: string): unknown {
  try { return JSON.parse(value); } catch { throw new NekiroApiError(0, 'NeKiro ' + field + ' is not valid JSON.', 'INVALID_RESPONSE'); }
}

export function buildAgentCard(input: AgentCardInput): AgentCardV02 {
  if (!['none', 'api_key', 'http_bearer', 'oauth2_client_credentials', 'mutual_tls'].includes(input.authentication)) throw new Error('authentication type is invalid');
  const agentId = readIdentifier(input.agentId, 'agentId');
  const permissions = input.permissions.map((permission, index) => {
    assertPermissionKeys(permission, index);
    return {
      id: readIdentifier(permission.id, 'permissions[' + index + '].id'),
      description: readText(permission.description, 'permissions[' + index + '].description', 1000),
    };
  });
  ensureUnique(permissions.map((permission) => permission.id), 'permission id');

  const declaredPermissions = new Set(permissions.map((permission) => permission.id));
  const parsed = parseCapabilities(input.capabilitiesJson);
  const seenSkillIds = new Set<string>();
  const skills = parsed.map((capability, index) => {
    assertAllowedKeys(capability, ['id', 'name', 'description', 'inputSchema', 'outputSchema', 'requiredPermissions'], 'capabilities[' + index + ']');
    const id = readIdentifier(capability.id, 'capabilities[' + index + '].id');
    if (seenSkillIds.has(id)) {
      throw new Error('duplicate capability id: ' + id);
    }
    seenSkillIds.add(id);
    const requiredPermissions = readStringArray(capability.requiredPermissions, 'capabilities[' + index + '].requiredPermissions');
    for (const permissionId of requiredPermissions) {
      if (!declaredPermissions.has(permissionId)) {
        throw new Error('required permission is not declared in permissions: ' + permissionId);
      }
    }

    return {
      id,
      name: readText(capability.name, 'capabilities[' + index + '].name', 120),
      description: readText(capability.description, 'capabilities[' + index + '].description', 2000),
      inputSchema: readJsonObject(capability.inputSchema, 'capabilities[' + index + '].inputSchema'),
      outputSchema: readJsonObject(capability.outputSchema, 'capabilities[' + index + '].outputSchema'),
      requiredPermissions,
    } satisfies AgentSkill;
  });

  const endpoint = readText(input.endpoint, 'endpoint', 2048);
  let endpointUrl: URL;
  try {
    endpointUrl = new URL(endpoint);
  } catch {
    throw new Error('endpoint must be an absolute http or https URL');
  }
  if (!['http:', 'https:'].includes(endpointUrl.protocol)) {
    throw new Error('endpoint must use http or https');
  }
  if (endpointUrl.username || endpointUrl.password || hasUriUserinfo(endpoint)) {
    throw new Error('endpoint must not contain userinfo credentials');
  }

  const version = readText(input.version, 'version');
  if (!isSemver(version)) {
    throw new Error('version must be strict SemVer');
  }
  validateAgentLimits(input.limits);

  return {
    schemaVersion: '0.2',
    agentId,
    name: readText(input.name, 'name', 120),
    description: readText(input.description, 'description', 4000),
    owner: {
      id: readIdentifier(input.ownerId, 'ownerId'),
      displayName: readText(input.ownerDisplayName, 'ownerDisplayName', 120),
    },
    version,
    protocol: {
      type: 'a2a',
      version: '0.3.0',
      transport: 'JSONRPC',
      endpoint,
    },
    skills,
    authentication: {type: input.authentication},
    permissions,
    limits: input.limits,
  };
}

function validateAgentLimits(value: AgentCardV02['limits']): void {
  if (!isRecord(value)) throw new Error('limits must be a JSON object');
  assertAllowedKeys(value, ['timeoutMs', 'maxInputBytes', 'maxOutputBytes', 'streaming'], 'limits');
  if (!Number.isInteger(value.timeoutMs) || value.timeoutMs < 1 || value.timeoutMs > 600000) throw new Error('limits.timeoutMs must be between 1 and 600000');
  if (!Number.isInteger(value.maxInputBytes) || value.maxInputBytes < 1) throw new Error('limits.maxInputBytes must be a positive integer');
  if (!Number.isInteger(value.maxOutputBytes) || value.maxOutputBytes < 1) throw new Error('limits.maxOutputBytes must be a positive integer');
  if (typeof value.streaming !== 'boolean') throw new Error('limits.streaming must be a boolean');
}

function assertPermissionKeys(value: unknown, index: number): void {
  if (!isRecord(value)) throw new Error('permissions[' + index + '] must be a JSON object');
  assertAllowedKeys(value, ['id', 'description'], 'permissions[' + index + ']');
}

function validateCatalogEntry(value: unknown, publicAgentOrigin = ''): CatalogEntry {
  const record = requireRecord(value, 'Catalog entry');
  assertAllowedKeys(record, ['card', 'publicationStatus', 'registeredAt', 'publishedAt', 'publicAgentId', 'publicUrl'], 'Catalog entry');
  const result: CatalogEntry = {
    card: validateCatalogCard(record.card),
    publicationStatus: requireEnum(record.publicationStatus, ['draft', 'published', 'disabled'], 'publicationStatus') as PublicationStatus,
    registeredAt: requireDateValue(record.registeredAt, 'registeredAt'),
  };
  if ('publishedAt' in record) result.publishedAt = requireDateValue(record.publishedAt, 'publishedAt');
  if (('publicAgentId' in record) !== ('publicUrl' in record)) throw new Error('Catalog public identity fields must be paired');
  if ('publicAgentId' in record) {
    result.publicAgentId = readPublicAgentID(record.publicAgentId);
    result.publicUrl = requirePublicAgentURL(record.publicUrl, result.publicAgentId, publicAgentOrigin);
  }
  return result;
}

function validatePublicAgentShare(value: unknown, expectedPublicAgentID: string, publicAgentOrigin: string): PublicAgentShare {
  const record = requireRecord(value, 'Public Agent share');
  assertAllowedKeys(record, ['schemaVersion', 'publicAgentId', 'publicUrl', 'registeredAt', 'availability', 'releases'], 'Public Agent share');
  const publicAgentID = readPublicAgentID(record.publicAgentId);
  if (publicAgentID !== expectedPublicAgentID) throw new Error('Public Agent identity changed');
  const releases = readRecordArray(record.releases, 'releases').map((release, index) => validatePublicAgentRelease(release, index));
  const availability = requireEnum(record.availability, ['installable', 'not_installable'], 'availability') as PublicAgentShare['availability'];
  if ((availability === 'installable') !== (releases.length > 0)) throw new Error('Public Agent availability does not match Releases');
  return {
    schemaVersion: record.schemaVersion === '1' ? '1' : (() => { throw new Error('Public Agent share schemaVersion is invalid'); })(),
    publicAgentId: publicAgentID,
    publicUrl: requirePublicAgentURL(record.publicUrl, publicAgentID, publicAgentOrigin),
    registeredAt: requireDateValue(record.registeredAt, 'registeredAt'),
    availability,
    releases,
  };
}

function validatePublicAgentRelease(value: Record<string, unknown>, index: number): PublicAgentRelease {
  const field = `releases[${index}]`;
  assertAllowedKeys(value, ['releaseId', 'agentId', 'name', 'description', 'owner', 'agentCardVersion', 'cardDigest', 'publishedAt', 'authenticationType', 'skills', 'permissions', 'limits'], field);
  const owner = requireRecord(value.owner, `${field}.owner`);
  assertAllowedKeys(owner, ['id', 'displayName'], `${field}.owner`);
  const skills = readRecordArray(value.skills, `${field}.skills`).map((skill, skillIndex) => {
    const skillField = `${field}.skills[${skillIndex}]`;
    assertAllowedKeys(skill, ['id', 'name', 'description', 'inputSchema', 'outputSchema', 'requiredPermissions'], skillField);
    return {
      id: readIdentifier(skill.id, `${skillField}.id`),
      name: readText(skill.name, `${skillField}.name`, 120),
      description: readText(skill.description, `${skillField}.description`, 2000),
      inputSchema: requireRecord(skill.inputSchema, `${skillField}.inputSchema`),
      outputSchema: requireRecord(skill.outputSchema, `${skillField}.outputSchema`),
      requiredPermissions: readStringArray(skill.requiredPermissions, `${skillField}.requiredPermissions`),
    };
  });
  const permissions = readRecordArray(value.permissions, `${field}.permissions`).map((permission, permissionIndex) => {
    assertPermissionKeys(permission, permissionIndex);
    return {id: readIdentifier(permission.id, `${field}.permissions[${permissionIndex}].id`), description: readText(permission.description, `${field}.permissions[${permissionIndex}].description`, 1000)};
  });
  validateAgentLimits(value.limits as AgentCardV02['limits']);
  return {
    releaseId: readIdentifier(value.releaseId, `${field}.releaseId`),
    agentId: readIdentifier(value.agentId, `${field}.agentId`),
    name: readText(value.name, `${field}.name`, 120),
    description: readText(value.description, `${field}.description`, 4000),
    owner: {id: readIdentifier(owner.id, `${field}.owner.id`), displayName: readText(owner.displayName, `${field}.owner.displayName`, 120)},
    agentCardVersion: requireSemver(value.agentCardVersion, `${field}.agentCardVersion`),
    cardDigest: requireDigest(value.cardDigest, `${field}.cardDigest`),
    publishedAt: requireDateValue(value.publishedAt, `${field}.publishedAt`),
    authenticationType: requireEnum(value.authenticationType, ['none', 'api_key', 'http_bearer', 'oauth2_client_credentials', 'mutual_tls'], `${field}.authenticationType`) as AuthenticationType,
    skills,
    permissions,
    limits: value.limits as AgentCardV02['limits'],
  };
}

function validateCatalogSearchResponse(value: unknown, publicAgentOrigin = ''): CatalogSearchResponse {
  const record = requireRecord(value, 'Catalog search response');
  assertAllowedKeys(record, ['items', 'nextCursor'], 'Catalog search response');
  if (!Array.isArray(record.items)) throw new Error('Catalog search items must be an array');
  const result: CatalogSearchResponse = {items: record.items.map((item) => validateCatalogEntry(item, publicAgentOrigin))};
  if ('nextCursor' in record) result.nextCursor = readText(record.nextCursor, 'nextCursor');
  return result;
}

function validateCatalogCard(value: unknown): AgentCardV02 {
  const record = requireRecord(value, 'Agent Card');
  assertAllowedKeys(record, ['schemaVersion', 'agentId', 'name', 'description', 'owner', 'version', 'protocol', 'skills', 'authentication', 'permissions', 'limits'], 'Agent Card');
  if (record.schemaVersion !== '0.2') throw new Error('Agent Card schemaVersion is invalid');
  const owner = requireRecord(record.owner, 'Agent Card owner');
  assertAllowedKeys(owner, ['id', 'displayName'], 'Agent Card owner');
  const protocol = requireRecord(record.protocol, 'Agent Card protocol');
  assertAllowedKeys(protocol, ['type', 'version', 'transport', 'endpoint'], 'Agent Card protocol');
  if (protocol.type !== 'a2a' || protocol.version !== '0.3.0' || protocol.transport !== 'JSONRPC') throw new Error('Agent Card protocol is invalid');
  const authentication = requireRecord(record.authentication, 'Agent Card authentication');
  assertAllowedKeys(authentication, ['type'], 'Agent Card authentication');
  const permissions = readRecordArray(record.permissions, 'permissions').map((permission, index) => {
    assertPermissionKeys(permission, index);
    return {id: readIdentifier(permission.id, `permissions[${index}].id`), description: readText(permission.description, `permissions[${index}].description`, 1000)};
  });
  ensureUnique(permissions.map((permission) => permission.id), 'permission id');
  const declaredPermissions = new Set(permissions.map((permission) => permission.id));
  const skillRecords = readRecordArray(record.skills, 'skills');
  if (skillRecords.length === 0) throw new Error('skills must contain at least one skill');
  const skillIds = new Set<string>();
  const skills = skillRecords.map((skill, index) => {
    assertAllowedKeys(skill, ['id', 'name', 'description', 'inputSchema', 'outputSchema', 'requiredPermissions'], `skills[${index}]`);
    const id = readIdentifier(skill.id, `skills[${index}].id`);
    if (skillIds.has(id)) throw new Error('duplicate skill id: ' + id);
    skillIds.add(id);
    const requiredPermissions = readStringArray(skill.requiredPermissions, `skills[${index}].requiredPermissions`);
    for (const permissionId of requiredPermissions) {
      if (!declaredPermissions.has(permissionId)) {
        throw new Error('required permission is not declared in permissions: ' + permissionId);
      }
    }
    return {
      id,
      name: readText(skill.name, `skills[${index}].name`, 120),
      description: readText(skill.description, `skills[${index}].description`, 2000),
      inputSchema: requireRecord(skill.inputSchema, `skills[${index}].inputSchema`),
      outputSchema: requireRecord(skill.outputSchema, `skills[${index}].outputSchema`),
      requiredPermissions,
    };
  });
  validateAgentLimits(record.limits as AgentCardV02['limits']);
  return {
    schemaVersion: '0.2',
    agentId: readIdentifier(record.agentId, 'agentId'),
    name: readText(record.name, 'name', 120),
    description: readText(record.description, 'description', 4000),
    owner: {id: readIdentifier(owner.id, 'owner.id'), displayName: readText(owner.displayName, 'owner.displayName', 120)},
    version: requireSemver(record.version, 'version'),
    protocol: {type: 'a2a', version: '0.3.0', transport: 'JSONRPC', endpoint: requireHttpUri(readText(protocol.endpoint, 'protocol.endpoint', 2048), 'protocol.endpoint')},
    skills,
    authentication: {type: requireEnum(authentication.type, ['none', 'api_key', 'http_bearer', 'oauth2_client_credentials', 'mutual_tls'], 'authentication.type') as AuthenticationType},
    permissions,
    limits: record.limits as AgentCardV02['limits'],
  };
}

function validateWorkspace(value: unknown): Workspace {
  const record = requireRecord(value, 'Workspace');
  assertAllowedKeys(record, ['workspaceId', 'ownerId', 'createdAt', 'updatedAt'], 'Workspace');
  return {
    workspaceId: readIdentifier(record.workspaceId, 'workspaceId'),
    ownerId: readIdentifier(record.ownerId, 'ownerId'),
    createdAt: requireDateValue(record.createdAt, 'createdAt'),
    updatedAt: requireDateValue(record.updatedAt, 'updatedAt'),
  };
}

function readRecordArray(value: unknown, field: string): Record<string, unknown>[] {
  if (!Array.isArray(value) || !value.every((item) => isRecord(item))) throw new Error(field + ' must be an array of JSON objects');
  return value;
}

export function mapCatalogEntry(entry: CatalogEntry): Agent {
  return {
    id: entry.card.agentId,
    name: entry.card.name,
    version: entry.card.version,
    owner: entry.card.owner.displayName,
    ownerId: entry.card.owner.id,
    description: entry.card.description,
    tags: entry.card.skills.map((skill) => skill.id),
    status: entry.publicationStatus,
    schema: JSON.stringify(entry.card, null, 2),
    permissions: entry.card.permissions,
    registeredAt: entry.registeredAt,
    publishedAt: entry.publishedAt,
    publicAgentId: entry.publicAgentId,
    publicUrl: entry.publicUrl,
  };
}

function validateReleaseProvenance(releaseId: unknown, cardDigest: unknown, field: string): void {
  if ((releaseId === undefined) !== (cardDigest === undefined)) throw new Error(`${field} Release provenance must contain both fields or neither`);
  if (releaseId !== undefined) {
    requireIdentifier(releaseId, field + ' agentReleaseId');
    requireDigest(cardDigest, field + ' agentCardDigest');
  }
}

function validateInstallation(value: unknown, workspaceId: string): Installation {
  const record = requireRecord(value, 'Installation');
  assertAllowedKeys(record, ['installationId', 'workspaceId', 'agentId', 'versionConstraint', 'installedVersion', 'installedReleaseId', 'acceptedPermissions', 'status', 'installedAt', 'updatedAt', 'uninstalledAt'], 'Installation');
  if (record.workspaceId !== workspaceId) throw new Error('Installation Workspace does not match the request');
  const installationID = readIdentifier(record.installationId, 'installationId');
  const agentID = readIdentifier(record.agentId, 'agentId');
  const versionConstraint = readText(record.versionConstraint, 'versionConstraint');
  const installedVersion = requireSemver(record.installedVersion, 'installedVersion');
  if (!satisfiesSemverRange(installedVersion, versionConstraint)) throw new Error('installedVersion does not satisfy versionConstraint');
  const acceptedPermissions = readStringArray(record.acceptedPermissions, 'acceptedPermissions').map((permission, index) => readIdentifier(permission, `acceptedPermissions[${index}]`));
  if ([...acceptedPermissions].sort().join('\u0000') !== acceptedPermissions.join('\u0000')) throw new Error('acceptedPermissions must be sorted');
  const status = requireEnum(record.status, ['enabled', 'disabled', 'uninstalled'], 'Installation status') as InstallationStatus;
  const installedAt = requireDateValue(record.installedAt, 'installedAt');
  const updatedAt = requireDateValue(record.updatedAt, 'updatedAt');
  if (Date.parse(installedAt) > Date.parse(updatedAt)) throw new Error('Installation updatedAt must not precede installedAt');
  const result: Installation = {
    installationId: installationID,
    workspaceId,
    agentId: agentID,
    versionConstraint,
    installedVersion,
    acceptedPermissions,
    status,
    installedAt,
    updatedAt,
  };
  if ('installedReleaseId' in record) result.installedReleaseId = readIdentifier(record.installedReleaseId, 'installedReleaseId');
  if (status === 'uninstalled') {
    if (!('uninstalledAt' in record)) throw new Error('uninstalled Installation requires uninstalledAt');
    const uninstalledAt = requireDateValue(record.uninstalledAt, 'uninstalledAt');
    if (Date.parse(uninstalledAt) !== Date.parse(updatedAt)) throw new Error('uninstalledAt must equal updatedAt');
    result.uninstalledAt = uninstalledAt;
  } else if ('uninstalledAt' in record) {
    throw new Error('active Installation must not contain uninstalledAt');
  }
  return result;
}

function validateInstallationLifecycleResponse(value: Installation, previous: Installation): Installation {
  const samePermissions = value.acceptedPermissions.length === previous.acceptedPermissions.length
    && value.acceptedPermissions.every((permission, index) => permission === previous.acceptedPermissions[index]);
  if (value.installationId !== previous.installationId
    || value.workspaceId !== previous.workspaceId
    || value.agentId !== previous.agentId
    || value.versionConstraint !== previous.versionConstraint
    || value.installedVersion !== previous.installedVersion
    || !samePermissions
    || value.installedReleaseId !== previous.installedReleaseId) {
    throw new NekiroApiError(200, 'NeKiro Installation lifecycle response changed immutable pin fields.', 'INVALID_RESPONSE');
  }
  return value;
}

function validateInstallationList(value: unknown, workspaceId: string): InstallationList {
  const record = requireRecord(value, 'Installation list');
  assertAllowedKeys(record, ['items', 'nextCursor'], 'Installation list');
  if (!Array.isArray(record.items)) throw new Error('Installation list items must be an array');
  const result: InstallationList = {
    items: record.items.map((item) => validateInstallation(item, workspaceId)),
  };
  if ('nextCursor' in record) result.nextCursor = readText(record.nextCursor, 'nextCursor');
  return result;
}

export function toPlatformErrorView(error: unknown, fallbackMessage: string): PlatformErrorView {
  if (error instanceof NekiroApiError) {
    return error.toView();
  }
  return {
    status: 0,
    code: 'CLIENT_ERROR',
    message: fallbackMessage,
  };
}

export function validateTrustedInstallation(value: Installation, release: AgentRelease, expected: {
  workspaceId: string;
  agentId: string;
  versionConstraint: string;
  acceptedPermissions: string[];
}): Installation {
  const samePermissions = value.acceptedPermissions.length === expected.acceptedPermissions.length
    && value.acceptedPermissions.every((permission, index) => permission === expected.acceptedPermissions[index]);
  if (value.workspaceId !== expected.workspaceId
    || value.agentId !== expected.agentId
    || value.versionConstraint !== expected.versionConstraint
    || value.installedVersion !== release.agentCardVersion
    || value.installedReleaseId !== release.releaseId
    || !samePermissions
    || value.status !== 'enabled') {
    throw new NekiroApiError(200, 'NeKiro Installation did not preserve the preflight Release identity.', 'INVALID_RESPONSE');
  }
  return value;
}

export function validatePublicInstallation(value: Installation, release: PublicAgentRelease, workspaceId: string, acceptedPermissions: string[]): Installation {
  const samePermissions = value.acceptedPermissions.length === acceptedPermissions.length
    && value.acceptedPermissions.every((permission, index) => permission === acceptedPermissions[index]);
  if (value.workspaceId !== workspaceId
    || value.agentId !== release.agentId
    || value.versionConstraint !== release.agentCardVersion
    || value.installedVersion !== release.agentCardVersion
    || value.installedReleaseId !== release.releaseId
    || !samePermissions
    || value.status !== 'enabled') {
    throw new NekiroApiError(200, 'NeKiro Installation did not preserve the selected public Release identity.', 'INVALID_RESPONSE');
  }
  return value;
}

function parseCapabilities(value: string): Record<string, unknown>[] {
  let parsed: unknown;
  try {
    parsed = JSON.parse(value);
  } catch {
    throw new Error('capabilitiesJson must be valid JSON');
  }
  if (!isRecord(parsed) || !Array.isArray(parsed.capabilities) || parsed.capabilities.length === 0) {
    throw new Error('capabilitiesJson must contain a non-empty capabilities array');
  }
  if (!parsed.capabilities.every(isRecord)) {
    throw new Error('every capability must be a JSON object');
  }
  return parsed.capabilities;
}

function readText(value: unknown, field: string, maxLength?: number): string {
  if (typeof value !== 'string' || value.trim() === '' || value !== value.trim() || (maxLength !== undefined && value.length > maxLength)) {
    throw new Error(field + ' must be a non-empty string');
  }
  return value;
}

function readIdentifier(value: unknown, field: string): string {
  const text = readText(value, field);
  if (!/^[A-Za-z0-9](?:[A-Za-z0-9._:-]{0,127})$/.test(text)) {
    throw new Error(field + ' must be a NeKiro safe identifier');
  }
  return text;
}

function readPublicAgentID(value: unknown): string {
  const text = readText(value, 'publicAgentId');
  if (!/^agt_[0-9a-f]{32}$/.test(text)) throw new Error('publicAgentId must be an exact public Agent identifier');
  return text;
}

function requirePublicAgentURL(value: unknown, publicAgentID: string, origin: string): string {
  const text = readText(value, 'publicUrl', 2048);
  if (typeof origin !== 'string' || origin === '' || origin !== origin.trim()) throw new Error('VITE_NEKIRO_PUBLIC_AGENT_ORIGIN is required');
  const expected = origin + '/a/' + publicAgentID;
  if (text !== expected) throw new Error('publicUrl is not the canonical configured public Agent URL');
  return text;
}

function isIpHostname(hostname: string): boolean {
  return /^\d{1,3}(?:\.\d{1,3}){3}$/.test(hostname) || hostname.includes(':');
}

function hasUriUserinfo(value: string): boolean {
  const authority = /^[A-Za-z][A-Za-z0-9+.-]*:\/\/([^/?#]*)/.exec(value)?.[1];
  return authority?.includes('@') ?? false;
}

function satisfiesSemverRange(version: string, range: string): boolean {
  // Masterminds/semver accepts backend aliases and partial prerelease tokens;
  // keep those syntax adapters narrow and delegate comparison semantics to the
  // maintained npm parser, which rejects non-canonical numeric components.
  const normalizedNumbers = normalizeLargeSemverNumbers(version, range);
  if (!normalizedNumbers || semverValid(normalizedNumbers.version) === null) return false;
  version = normalizedNumbers.version;
  range = normalizedNumbers.range;
  if (range.length > 512) return false;
  const branches = range.split('||');
  if (branches.length > 32) return false;
  const parsedBranches = branches.map((branch) => parseSemverBranch(branch));
  if (parsedBranches.some((branch) => branch === undefined)) return false;
  return parsedBranches.some((branch) => branch !== undefined && satisfiesSemverBranch(version, branch));
}

const MAX_UINT64 = 18446744073709551615n;
const MAX_SAFE_INTEGER_BIGINT = BigInt(Number.MAX_SAFE_INTEGER);

interface SemverTokenParts {
  prefix: string;
  major: string;
  minor?: string;
  patch?: string;
  prerelease?: string;
  build?: string;
}

interface SemverNumericMaps {
  core: [Map<string, string>, Map<string, string>, Map<string, string>];
  prerelease: Map<number, Map<string, string>>;
}

function normalizeLargeSemverNumbers(version: string, range: string): {version: string; range: string} | undefined {
  const normalizedRange = range.replace(/=>/g, '>=').replace(/=</g, '<=').replace(/~>/g, '~');
  const maps: SemverNumericMaps = {core: [new Map(), new Map(), new Map()], prerelease: new Map()};
  if (!collectSemverTokenNumbers(version, maps) || !collectSemverTokenNumbers(normalizedRange, maps)) return undefined;
  maps.core.forEach((map) => map.set('0', '0'));
  const needsCoreMapping = maps.core.map((map) => finalizeSemverNumberMap(map)).some(Boolean);
  const needsPrereleaseMapping = [...maps.prerelease.values()].map((map) => finalizeSemverNumberMap(map)).some(Boolean);
  const needsMapping = needsCoreMapping || needsPrereleaseMapping;
  if (!needsMapping) return {version, range};
  return {
    version: replaceLargeSemverTokenNumbers(version, maps),
    range: replaceLargeSemverTokenNumbers(normalizePartialPrerelease(normalizedRange), maps),
  };
}

function collectSemverTokenNumbers(value: string, maps: SemverNumericMaps): boolean {
  const tokenPattern = /(^|[\s,|])((?:>=|<=|!=|=>|=<|>|<|=|~>|~|\^)?)(v?(?:\d+|[xX*])(?:\.(?:\d+|[xX*]))?(?:\.(?:\d+|[xX*]))?(?:-[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?(?:\+[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?)(?=$|[\s,|])/g;
  let match: RegExpExecArray | null;
  while ((match = tokenPattern.exec(value)) !== null) {
    const token = parseSemverTokenParts(match[3]);
    if (!token) continue;
    for (const [index, component] of [token.major, token.minor, token.patch].entries()) {
      if (component === undefined || isSemverWildcard(component)) continue;
      if (!recordSemverNumber(component, maps.core[index])) return false;
    }
    if (token.prerelease) {
      for (const [index, identifier] of token.prerelease.split('.').entries()) {
        if (/^0\d/.test(identifier)) return false;
        if (/^\d+$/.test(identifier) && !recordSemverNumber(identifier, getPrereleaseMap(maps, index))) return false;
      }
    }
  }
  return true;
}

function recordSemverNumber(value: string, map: Map<string, string>): boolean {
  if (/^0\d/.test(value)) return false;
  try {
    const numeric = BigInt(value);
    if (numeric > MAX_UINT64) return false;
  } catch {
    return false;
  }
  map.set(value, value);
  return true;
}

function finalizeSemverNumberMap(map: Map<string, string>): boolean {
  const values = [...map.keys()];
  const needsMapping = values.some((value) => BigInt(value) >= MAX_SAFE_INTEGER_BIGINT);
  if (!needsMapping) return false;
  values.sort((left, right) => (BigInt(left) < BigInt(right) ? -1 : BigInt(left) > BigInt(right) ? 1 : 0));
  values.forEach((value, index) => map.set(value, String(index)));
  return true;
}

function replaceLargeSemverTokenNumbers(value: string, maps: SemverNumericMaps): string {
  const tokenPattern = /(^|[\s,|])((?:>=|<=|!=|=>|=<|>|<|=|~>|~|\^)?)(v?(?:\d+|[xX*])(?:\.(?:\d+|[xX*]))?(?:\.(?:\d+|[xX*]))?(?:-[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?(?:\+[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?)(?=$|[\s,|])/g;
  return value.replace(tokenPattern, (_match, prefix: string, operator: string, tokenValue: string) => prefix + operator + renderSemverToken(tokenValue, maps));
}

function renderSemverToken(value: string, maps: SemverNumericMaps): string {
  const token = parseSemverTokenParts(value);
  if (!token) return value;
  const core = [token.major, token.minor, token.patch].map((component, index) => component === undefined || isSemverWildcard(component) ? component : maps.core[index].get(component) ?? component);
  const prerelease = token.prerelease?.split('.').map((identifier, index) => /^\d+$/.test(identifier) ? getPrereleaseMap(maps, index).get(identifier) ?? identifier : identifier).join('.');
  return token.prefix + core[0] + (core[1] === undefined ? '' : '.' + core[1]) + (core[2] === undefined ? '' : '.' + core[2]) + (prerelease === undefined ? '' : '-' + prerelease) + (token.build === undefined ? '' : '+' + token.build);
}

function parseSemverTokenParts(value: string): SemverTokenParts | undefined {
  const match = /^(v?)(\d+|[xX*])(?:\.(\d+|[xX*]))?(?:\.(\d+|[xX*]))?(?:-([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?(?:\+([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?$/.exec(value);
  if (!match) return undefined;
  return {prefix: match[1], major: match[2], minor: match[3], patch: match[4], prerelease: match[5], build: match[6]};
}

function getPrereleaseMap(maps: SemverNumericMaps, index: number): Map<string, string> {
  const existing = maps.prerelease.get(index);
  if (existing) return existing;
  const created = new Map<string, string>();
  maps.prerelease.set(index, created);
  return created;
}

interface SemverRangeBranch {
  baseRange: string;
  exclusions: string[];
}

function satisfiesSemverBranch(version: string, branch: SemverRangeBranch): boolean {
  if (!semverSatisfies(version, branch.baseRange)) return false;
  return branch.exclusions.every((exclusion) => !semverSatisfies(version, exclusion));
}

function parseSemverBranch(branch: string): SemverRangeBranch | undefined {
  const normalized = normalizeSemverRangeSyntax(branch);
  if (!normalized || normalized.split(',').some((part) => part.trim() === '')) return undefined;
  const rawTokens = normalized.split(/[\s,]+/).filter(Boolean);
  if (rawTokens.length === 0) return undefined;
  const baseTokens: string[] = [];
  const exclusions: string[] = [];
  for (let index = 0; index < rawTokens.length; index += 1) {
    let token = rawTokens[index];
    if (token === '!=') {
      token = rawTokens[++index] ?? '';
      if (!token) return undefined;
      const exclusion = semverExclusionRange(token);
      if (!exclusion) return undefined;
      exclusions.push(exclusion.range);
      continue;
    }
    if (token.startsWith('!=')) {
      const exclusion = semverExclusionRange(token.slice(2));
      if (!exclusion) return undefined;
      exclusions.push(exclusion.range);
      continue;
    }
    baseTokens.push(token);
  }
  const baseRange = semverValidRange(normalizePartialPrerelease(baseTokens.join(' ') || '*'), {loose: false});
  if (baseRange === null) return undefined;
  return {baseRange, exclusions};
}

function normalizeSemverRangeSyntax(value: string): string | undefined {
  const trimmed = value.trim();
  if (trimmed.length === 0) return undefined;
  return trimmed.replace(/=>/g, '>=').replace(/=</g, '<=').replace(/~>/g, '~');
}

interface SemverExclusion {
  range: string;
}

function semverExclusionRange(value: string): SemverExclusion | undefined {
  const normalized = normalizePartialPrerelease(value);
  const match = /^v?(\d+|[xX*])(?:\.(\d+|[xX*]))?(?:\.(\d+|[xX*]))?(?:-([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?(?:\+[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?$/.exec(normalized);
  if (!match || semverValidRange(normalized, {loose: false}) === null) return undefined;
  const majorWildcard = isSemverWildcard(match[1]);
  const minorWildcard = match[2] === undefined || isSemverWildcard(match[2]);
  const patchWildcard = match[3] === undefined || isSemverWildcard(match[3]);
  if (majorWildcard) return {range: '*'};
  const major = Number(match[1]);
  const minor = minorWildcard ? 0 : Number(match[2]);
  const patch = patchWildcard ? 0 : Number(match[3]);
  const range = minorWildcard
      ? `>=${major}.0.0 <${major + 1}.0.0`
      : patchWildcard
        ? `>=${major}.${minor}.0 <${major}.${minor + 1}.0`
      : normalized;
  return {range};
}

function normalizePartialPrerelease(value: string): string {
  return value.replace(/(^|[\s,])((?:>=|<=|!=|=>|=<|>|<|=|~>|~|\^)?)(v?)(\d+|[xX*])(?:\.(\d+|[xX*]))?(?:\.(\d+|[xX*]))?(-[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)(?=$|[\s,])/g, (_match, prefix: string, operator: string, versionPrefix: string, major: string, minor: string | undefined, patch: string | undefined, prerelease: string) => {
    return prefix + operator + versionPrefix + major + '.' + (minor ?? '0') + '.' + (patch ?? '0') + prerelease;
  });
}

function isSemverWildcard(value: string | undefined): boolean {
  return value === undefined || value === 'x' || value === 'X' || value === '*';
}

function readStringArray(value: unknown, field: string): string[] {
  if (!Array.isArray(value) || !value.every((item) => typeof item === 'string' && item.trim() !== '' && item === item.trim())) {
    throw new Error(field + ' must be an array of non-empty strings');
  }
  const result = value as string[];
  ensureUnique(result, field + ' value');
  return result;
}

function readJsonObject(value: unknown, field: string): JsonObject {
  if (!isRecord(value)) {
    throw new Error(field + ' must be a JSON object');
  }
  return value;
}

function ensureUnique(values: string[], label: string): void {
  const seen = new Set<string>();
  for (const value of values) {
    if (seen.has(value)) {
      throw new Error('duplicate ' + label + ': ' + value);
    }
    seen.add(value);
  }
}

function isSemver(value: string): boolean {
  return /^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:-(?:0|[1-9][0-9]*|[0-9]*[A-Za-z-][0-9A-Za-z-]*)(?:\.(?:0|[1-9][0-9]*|[0-9]*[A-Za-z-][0-9A-Za-z-]*))*)?(?:\+[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?$/.test(value);
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

function parseJson<T>(value: string): T | undefined {
  if (!value) {
    return undefined;
  }
  try {
    return JSON.parse(value) as T;
  } catch {
    return undefined;
  }
}

function isPlatformErrorV4(value: unknown): value is PlatformErrorV4 {
  if (!isRecord(value) || typeof value.code !== 'string' || !(value.code in PLATFORM_ERROR_MESSAGES) || value.message !== PLATFORM_ERROR_MESSAGES[value.code as PlatformErrorCode] || typeof value.traceId !== 'string' || !/^[A-Za-z0-9](?:[A-Za-z0-9._:-]{0,127})$/.test(value.traceId)) return false;
  const correlated = 'invocationId' in value || 'rootTaskId' in value;
  const allowed = correlated ? ['code', 'message', 'traceId', 'invocationId', 'rootTaskId'] : ['code', 'message', 'traceId'];
  if (Object.keys(value).some((key) => !allowed.includes(key))) return false;
  if (!correlated) return true;
  return typeof value.invocationId === 'string' && typeof value.rootTaskId === 'string' && /^[A-Za-z0-9](?:[A-Za-z0-9._:-]{0,127})$/.test(value.invocationId) && /^[A-Za-z0-9](?:[A-Za-z0-9._:-]{0,127})$/.test(value.rootTaskId);
}
