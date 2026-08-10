# Instance Registration v1

## Ownership

`registry.InstanceRegistrar` publishes ephemeral topology for the Runtime that
owns one exact process endpoint. It does not publish Agent Cards or Releases,
authorize Workspace installations, select instances, invoke Agents, or write
Ledger records.

## API

```go
type InstanceRegistrar interface {
    Register(context.Context, Registration) (InstanceLease, error)
    Capabilities() Capabilities
    Close() error
}

type InstanceLease interface {
    Done() <-chan struct{}
    Err() error
    Close(context.Context) error
}
```

`Registration` contains one byte-exact `ReleaseTarget` and one immutable ready
`Instance`. Provider adapters may narrow the accepted endpoint shape. The
Nacos v1 adapter requires exactly one canonical IPv4 or IPv6 TCP endpoint for
its configured port name.

## Lifecycle

```text
Register request -> active lease -> periodic heartbeat
                              |-> heartbeat failure -> terminal
                              |-> Close -> deregister once -> closed
```

- Initial registration uses exactly one request.
- Each heartbeat uses exactly one request at the configured interval.
- The first heartbeat failure permanently closes `Done` and latches a typed
  error in `Err`.
- `Close` is idempotent and performs at most one deregistration request.
- Registrar closure closes all accepted leases and reports deregistration
  failures.
- There is no retry, reconnect, provider switch, alternate binding, cached
  success, or stale lease fallback.

## Nacos Mapping

The adapter publishes an ephemeral instance to one explicit namespace, group,
service, and cluster. Weight, heartbeat interval, heartbeat timeout, IP delete
timeout, and the exact instance identity in `nekiro.instanceId` are sent on
registration and heartbeat. Caller-provided `preserved.*` metadata is rejected
so it cannot override the lease policy. Only safe instance metadata is copied.
The access token remains bootstrap-only.

HTTP 401/403 is `unauthorized`; cancellation is `canceled`; transport, rate,
and other non-success statuses are `unavailable`; malformed acknowledgements
are `invalid`; explicit lifecycle closure is `closed`.

## Runtime Requirement

A Runtime using this contract must fail startup when initial registration
fails and must stop readiness/serving when its lease terminates. A Runtime may
start a new registration only through a new explicit lifecycle; the lease does
not recover itself.
