// Package testkit supplies provider-neutral Config Center conformance checks.
// It is a test utility; providers supply all state changes and interruption
// controls explicitly through Harness rather than receiving production control
// APIs from this package.
package testkit

import (
	"context"
	"testing"
	"time"

	configcenter "github.com/NeKiro-project/NeKiro/config_center"
)

// Harness supplies one isolated provider fixture. Publisher, Publish, and
// Delete are optional as a group so a read-only adapter can still exercise the
// shared lifecycle contract. Interrupt is required and must deterministically
// terminally interrupt this fixture without adding a production control API.
type Harness struct {
	Reader    configcenter.DynamicConfiguration
	Publisher configcenter.ConfigurationPublisher
	Publish   func(context.Context, configcenter.Key, []byte) error
	Delete    func(context.Context, configcenter.Key) error
	Interrupt func(context.Context) error
	Cleanup   func() error
}

// Run validates the reusable non-close lifecycle contract. It leaves no
// recovery path: it ends by invoking the fixture's explicit terminal interrupt.
func Run(t testing.TB, harness Harness) {
	t.Helper()
	validateHarness(t, harness)
	defer cleanup(t, harness)

	key := mustKey(t, "conformance/alpha")
	missing, err := harness.Reader.Get(context.Background(), key)
	assertCode(t, err, configcenter.CodeMissing)
	if missing.Present() || missing.Content() != nil || missing.Revision().Scoped() {
		t.Fatalf("Get missing state = %#v", missing)
	}

	observation, err := harness.Reader.Observe(context.Background(), key)
	if err != nil {
		t.Fatalf("Observe missing: %v", err)
	}
	if observation.Initial.Present() || !observation.Initial.Revision().Scoped() || observation.Initial.Revision().Order() != 0 {
		t.Fatalf("Observe missing initial = %#v", observation.Initial)
	}

	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = observation.Subscription.Next(canceled)
	assertCode(t, err, configcenter.CodeCanceled)

	if harness.Publisher != nil {
		runPublisherConformance(t, harness, key, observation)
	} else {
		if err := observation.Subscription.Close(); err != nil {
			t.Fatalf("subscription Close: %v", err)
		}
		_, err := observation.Subscription.Next(context.Background())
		assertCode(t, err, configcenter.CodeSubscriptionClosed)
	}

	interrupted, err := harness.Reader.Observe(context.Background(), key)
	if err != nil {
		t.Fatalf("Observe before interruption: %v", err)
	}
	if err := harness.Interrupt(context.Background()); err != nil {
		t.Fatalf("fixture Interrupt: %v", err)
	}
	_, err = next(interrupted.Subscription)
	assertCode(t, err, configcenter.CodeWatchInterrupted)
	_, err = harness.Reader.Get(context.Background(), key)
	assertCode(t, err, configcenter.CodeWatchInterrupted)

	assertRevisionClassifications(t)
}

// RunReaderClose validates reader closure in a distinct fixture, since closure
// and watch interruption are both terminal by contract.
func RunReaderClose(t testing.TB, harness Harness) {
	t.Helper()
	validateHarness(t, harness)
	defer cleanup(t, harness)

	key := mustKey(t, "conformance/close")
	observation, err := harness.Reader.Observe(context.Background(), key)
	if err != nil {
		t.Fatalf("Observe before reader Close: %v", err)
	}
	if err := harness.Reader.Close(); err != nil {
		t.Fatalf("reader Close: %v", err)
	}
	_, err = observation.Subscription.Next(context.Background())
	assertCode(t, err, configcenter.CodeReaderClosed)
	_, err = harness.Reader.Get(context.Background(), key)
	assertCode(t, err, configcenter.CodeReaderClosed)
}

func runPublisherConformance(t testing.TB, harness Harness, key configcenter.Key, observation configcenter.Observation) {
	t.Helper()
	if _, readerIsPublisher := harness.Reader.(configcenter.ConfigurationPublisher); readerIsPublisher {
		t.Fatal("runtime-facing reader also exposes publisher authority")
	}

	empty := []byte{}
	if err := harness.Publish(context.Background(), key, empty); err != nil {
		t.Fatalf("Publish present-empty: %v", err)
	}
	emptyEvent, err := next(observation.Subscription)
	if err != nil {
		t.Fatalf("Next present-empty: %v", err)
	}
	assertTransition(t, observation.Initial.Revision(), emptyEvent, configcenter.EventUpdate, true, empty)

	input := []byte("published")
	if err := harness.Publish(context.Background(), key, input); err != nil {
		t.Fatalf("Publish value: %v", err)
	}
	input[0] = 'X'
	update, err := next(observation.Subscription)
	if err != nil {
		t.Fatalf("Next update: %v", err)
	}
	assertTransition(t, emptyEvent.Snapshot().Revision(), update, configcenter.EventUpdate, true, []byte("published"))
	returned := update.Snapshot().Content()
	returned[0] = 'Y'
	current, err := harness.Reader.Get(context.Background(), key)
	if err != nil || !current.Present() || current.Revision().Scoped() || string(current.Content()) != "published" {
		t.Fatalf("Get after copied publish = %#v, %v", current, err)
	}

	if err := harness.Delete(context.Background(), key); err != nil {
		t.Fatalf("Delete value: %v", err)
	}
	deletion, err := next(observation.Subscription)
	if err != nil {
		t.Fatalf("Next delete: %v", err)
	}
	assertTransition(t, update.Snapshot().Revision(), deletion, configcenter.EventDelete, false, nil)
	if err := harness.Delete(context.Background(), key); err == nil {
		t.Fatal("Delete missing succeeded")
	} else {
		assertCode(t, err, configcenter.CodeMissing)
	}

	if err := observation.Subscription.Close(); err != nil {
		t.Fatalf("subscription Close: %v", err)
	}
	_, err = observation.Subscription.Next(context.Background())
	assertCode(t, err, configcenter.CodeSubscriptionClosed)
}

func validateHarness(t testing.TB, harness Harness) {
	t.Helper()
	if harness.Reader == nil || harness.Interrupt == nil || harness.Cleanup == nil {
		t.Fatal("Harness requires Reader, Interrupt, and Cleanup")
	}
	publisherConfigured := harness.Publisher != nil
	callbacksConfigured := harness.Publish != nil || harness.Delete != nil
	if publisherConfigured != callbacksConfigured || (publisherConfigured && (harness.Publish == nil || harness.Delete == nil)) {
		t.Fatal("Harness publisher and Publish/Delete callbacks must be configured together")
	}
}

func cleanup(t testing.TB, harness Harness) {
	t.Helper()
	if err := harness.Cleanup(); err != nil {
		t.Errorf("fixture cleanup: %v", err)
	}
}

func mustKey(t testing.TB, value string) configcenter.Key {
	t.Helper()
	key, err := configcenter.ParseKey(value)
	if err != nil {
		t.Fatalf("ParseKey(%q): %v", value, err)
	}
	return key
}

func next(subscription configcenter.Subscription) (configcenter.ConfigurationEvent, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return subscription.Next(ctx)
}

func assertTransition(t testing.TB, previous configcenter.Revision, event configcenter.ConfigurationEvent, kind configcenter.EventKind, present bool, content []byte) {
	t.Helper()
	if event.Kind() != kind || event.Snapshot().Present() != present || !bytesEqual(event.Snapshot().Content(), content) {
		t.Fatalf("event = %#v, want kind=%q present=%v content=%q", event, kind, present, content)
	}
	if err := configcenter.ValidateNextRevision(previous, event.Snapshot().Revision()); err != nil {
		t.Fatalf("revision progression: %v", err)
	}
}

func assertRevisionClassifications(t testing.TB) {
	t.Helper()
	first := configcenter.NewObservationRevision()
	second, err := configcenter.AdvanceRevision(first)
	if err != nil {
		t.Fatal(err)
	}
	third, err := configcenter.AdvanceRevision(second)
	if err != nil {
		t.Fatal(err)
	}
	for _, testCase := range []struct {
		previous  configcenter.Revision
		candidate configcenter.Revision
		code      configcenter.Code
	}{
		{first, first, configcenter.CodeRevisionDuplicate},
		{second, first, configcenter.CodeRevisionStale},
		{first, third, configcenter.CodeRevisionGap},
		{first, configcenter.NewObservationRevision(), configcenter.CodeRevisionOutOfOrder},
	} {
		assertCode(t, configcenter.ValidateNextRevision(testCase.previous, testCase.candidate), testCase.code)
	}
}

func assertCode(t testing.TB, err error, want configcenter.Code) {
	t.Helper()
	if err == nil {
		t.Fatalf("error = nil, want %q", want)
	}
	got, ok := configcenter.CodeOf(err)
	if !ok || got != want {
		t.Fatalf("error code = %q (ok=%v), want %q; error=%v", got, ok, want, err)
	}
}

func bytesEqual(left, right []byte) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
