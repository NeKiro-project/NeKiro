export const INVOCATION_CONTRACT_ENDPOINTS = [
  'POST /v4/workspaces/{workspaceId}/invocations',
  'GET /v4/workspaces/{workspaceId}/invocations/{invocationId}',
  'GET /v4/workspaces/{workspaceId}/traces/{traceId}',
] as const;

export const LEDGER_CONTRACT_FACTS = [
  'Invocation Dispatch, A2A Router, and metadata-only Ledger are read through the public Gateway v4 routes.',
  'The Console renders only returned metadata and committed events; it never fabricates trace nodes or timeout events.',
  'Invocation input/output payloads are not persisted in the metadata-only Ledger.',
] as const;
