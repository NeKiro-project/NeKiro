package configcenter

import (
	"context"
	"testing"
)

type contractSubscription struct{}

func (contractSubscription) Next(context.Context) (ConfigurationEvent, error) {
	return ConfigurationEvent{}, NewError(CodeSubscriptionClosed, ErrorDetails{Operation: OperationNext})
}
func (contractSubscription) Close() error             { return nil }
func (contractSubscription) Stats() SubscriptionStats { return SubscriptionStats{} }

func TestSubscriptionInterfaceHasDistinctLifecycleTypes(t *testing.T) {
	var subscription Subscription = contractSubscription{}
	_, err := subscription.Next(context.Background())
	code, ok := CodeOf(err)
	if !ok || code != CodeSubscriptionClosed {
		t.Fatalf("subscription lifecycle error = %v", err)
	}
	stats := subscription.Stats()
	stats.SuppressedNotifications = 3
	if got := subscription.Stats().SuppressedNotifications; got != 0 {
		t.Fatalf("stats was not returned as a value: %d", got)
	}
	if err := subscription.Close(); err != nil {
		t.Fatalf("idempotent close contract fixture failed: %v", err)
	}
}
