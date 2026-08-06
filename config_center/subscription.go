package configcenter

import "context"

// Subscription delivers ordered transitions for one Observe call. A canceled
// Next wait leaves the subscription open; explicit subscription close, reader
// close, and watch interruption remain distinct typed outcomes.
type Subscription interface {
	Next(context.Context) (ConfigurationEvent, error)
	Close() error
	Stats() SubscriptionStats
}

// SubscriptionStats contains safe aggregate state only. It never includes
// configuration content or source implementation details.
type SubscriptionStats struct {
	SuppressedNotifications uint64
}
