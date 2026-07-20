# Data Model: A2A Router Foundation

## Router Config

Required, non-persisted deployment settings: listen address, Router service principals, Control Plane resolve URL, Control Plane service token, internal request body limit, Control Plane response body limit, and resolution deadline.

## Router Principal

Configured service identity: `id` and `tokenSha256`. The raw token is supplied by callers and never stored in config output or logs.

## Dispatch Envelope

Router Internal v3 request containing invocation/root/trace correlation, caller, Workspace, Agent, exact version, capability, input object, and stream mode.

## Resolved Target

Ephemeral Control Plane Internal v2 response: exact Agent Card plus resolved Installation facts. Router Foundation does not store it.
