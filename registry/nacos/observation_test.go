package nacos

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/NeKiro-project/NeKiro/registry"
)

type failingSubscriber struct{ err error }

func (failingSubscriber) Guarantees() NamingSubscriptionGuarantees {
	return validSubscriptionGuarantees()
}

func (subscriber failingSubscriber) Subscribe(context.Context, NamingSubscribeRequest) (NamingSubscription, error) {
	return NamingSubscription{}, subscriber.err
}

func TestNamingSubscriptionValidatesLifecycleAndCopiesPayload(t *testing.T) {
	stream := newFixturePushStream()
	subscription, err := NewNamingSubscription([]byte(initialServiceInfo), stream)
	if err != nil {
		t.Fatal(err)
	}

	payload := subscription.InitialPayload()
	payload[0] = '['
	if string(subscription.InitialPayload()) != initialServiceInfo {
		t.Fatal("InitialPayload did not return an isolated copy")
	}
	stream.events <- []byte("next")
	next, err := subscription.Next(context.Background())
	if err != nil || string(next) != "next" {
		t.Fatalf("Next=%q error=%v", next, err)
	}
	if err := subscription.Close(); err != nil {
		t.Fatal(err)
	}
	if err := subscription.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestNamingSubscriptionRejectsInvalidState(t *testing.T) {
	var typedNil *fixturePushStream
	tests := []struct {
		name    string
		initial []byte
		stream  []byte
	}{
		{name: "empty initial"},
		{name: "typed nil stream", stream: []byte(initialServiceInfo)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			candidate := NamingSubscription{}
			var err error
			if test.name == "typed nil stream" {
				candidate, err = NewNamingSubscription(test.stream, typedNil)
			} else {
				candidate, err = NewNamingSubscription(test.initial, newFixturePushStream())
			}
			if !errors.Is(err, registry.ErrInvalid) {
				t.Fatalf("NewNamingSubscription error=%v", err)
			}
			if _, err := candidate.Next(context.Background()); !errors.Is(err, registry.ErrInvalid) {
				t.Fatalf("zero subscription Next error=%v", err)
			}
			if err := candidate.Close(); err != nil {
				t.Fatalf("zero subscription Close error=%v", err)
			}
		})
	}
}

func TestObservationErrorNormalization(t *testing.T) {
	preserved := registry.NewOutcomeError(registry.OutcomeWatchInterrupted, registry.CauseWatchEventInvalid)
	if got := safeSubscribeError(preserved); !errors.Is(got, registry.ErrWatchInterrupted) {
		t.Fatalf("safeSubscribeError outcome=%v", got)
	}
	if got := safeSubscribeError(errors.New("provider detail")); !errors.Is(got, registry.ErrUnavailable) {
		t.Fatalf("safeSubscribeError plain=%v", got)
	}
	if got := safeWatchTerminal(preserved); !errors.Is(got, registry.ErrWatchInterrupted) {
		t.Fatalf("safeWatchTerminal preserved=%v", got)
	}
	for _, err := range []error{registry.NewOutcomeError(registry.OutcomeUnavailable, registry.CauseProviderUnavailable), errors.New("provider detail")} {
		got := safeWatchTerminal(err)
		var outcome *registry.OutcomeError
		if !errors.Is(got, registry.ErrWatchInterrupted) || !errors.As(got, &outcome) || outcome.Cause() != registry.CauseStreamEOF {
			t.Fatalf("safeWatchTerminal fallback=%v", got)
		}
	}
}

func TestObservationTopologyComparison(t *testing.T) {
	target := testTarget(t)
	binding, err := NewBinding(BindingInput{Target: target, ServiceName: "runtime-b", GroupName: "NEKIRO", ClusterName: "DEFAULT"})
	if err != nil {
		t.Fatal(err)
	}
	initial, err := snapshotFromPayload([]byte(initialServiceInfo), binding, "a2a", 0)
	if err != nil {
		t.Fatal(err)
	}
	changed, err := snapshotFromPayload([]byte(changedServiceInfo), binding, "a2a", 1)
	if err != nil {
		t.Fatal(err)
	}
	if !samePublicTopology(initial, initial) {
		t.Fatal("identical snapshots were not recognized")
	}
	if samePublicTopology(initial, changed) {
		t.Fatal("different snapshots were recognized as identical")
	}
}

func TestObserveClassifiesSubscriptionAndInitialPayloadFailures(t *testing.T) {
	target := testTarget(t)
	binding, err := NewBinding(BindingInput{Target: target, ServiceName: "runtime-b", GroupName: "NEKIRO", ClusterName: "DEFAULT"})
	if err != nil {
		t.Fatal(err)
	}

	directory := newObservationFixtureDirectory(t, binding, failingSubscriber{err: errors.New("provider detail")})
	if _, err := directory.Observe(t.Context(), target); !errors.Is(err, registry.ErrUnavailable) {
		t.Fatalf("Subscribe failure=%v", err)
	}

	subscriber := newFixtureSubscriber([]byte(strings.Repeat("x", 4097)))
	directory = newObservationFixtureDirectory(t, binding, subscriber)
	if _, err := directory.Observe(t.Context(), target); !errors.Is(err, registry.ErrInvalid) {
		t.Fatalf("oversized initial payload error=%v", err)
	}
}

func TestObservationWatchCloseTerminatesSource(t *testing.T) {
	target := testTarget(t)
	binding, err := NewBinding(BindingInput{Target: target, ServiceName: "runtime-b", GroupName: "NEKIRO", ClusterName: "DEFAULT"})
	if err != nil {
		t.Fatal(err)
	}
	subscriber := newFixtureSubscriber([]byte(initialServiceInfo))
	directory := newObservationFixtureDirectory(t, binding, subscriber)
	observation, err := directory.Observe(t.Context(), target)
	if err != nil {
		t.Fatal(err)
	}
	if err := observation.Watch().Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := observation.Watch().Next(t.Context()); !errors.Is(err, registry.ErrClosed) {
		t.Fatalf("Next after Close error=%v", err)
	}
	select {
	case <-subscriber.latest().done:
	default:
		t.Fatal("subscription source remained open")
	}
}
