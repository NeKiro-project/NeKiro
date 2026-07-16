# Router Foundation Contract Guide

This feature consumes existing contracts only:

- `contracts/openapi/router-internal.v3.yaml` for `POST /internal/v3/invocations`.
- `contracts/openapi/control-plane-internal.v2.yaml` for `/internal/v2/resolve-agent`.
- `contracts/schemas/platform-error.v4.schema.json` for Router Internal v3 errors.
- `contracts/schemas/platform-error.v3.schema.json` for Control Plane Internal v2 errors.

No shared contract file is changed by Spec 013. Router Foundation maps Control
Plane v3 error codes into Router v4 phase errors only when the request has
trusted dispatch correlation. Successful resolution returns correlated
`ROUTE_NOT_FOUND` until Agent transport is implemented.
