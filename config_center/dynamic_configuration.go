package configcenter

import "context"

// DynamicConfiguration is the read-only runtime-facing capability. It has no
// publishing methods, so receiving a reader never grants write authority.
type DynamicConfiguration interface {
	Get(context.Context, Key) (Snapshot, error)
	Observe(context.Context, Key) (Observation, error)
	Close() error
}

// Observation combines an initial immutable snapshot with a subscription that
// is already registered before Observe returns.
type Observation struct {
	Initial      Snapshot
	Subscription Subscription
}

// ConfigurationPublisher is the separately injected administrative write
// capability. It is deliberately not embedded in DynamicConfiguration.
type ConfigurationPublisher interface {
	Publish(context.Context, Key, []byte) error
	Delete(context.Context, Key) error
	Close() error
}
