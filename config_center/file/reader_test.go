package file

import (
	"context"
	"errors"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	configcenter "github.com/NeKiro-project/NeKiro/config_center"
)

func TestOpenReaderUsesOneExplicitRootWatch(t *testing.T) {
	root := t.TempDir()
	reader, err := OpenReader(ReaderConfig{Root: root, MaxPayloadBytes: 1024, SubscriptionBuffer: 2})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = reader.Close() }()
	watchList := reader.watcher.WatchList()
	if len(watchList) != 1 || filepath.Clean(watchList[0]) != filepath.Clean(root) {
		t.Fatalf("watch list = %#v, want exactly root %q", watchList, root)
	}
}

func TestOpenReaderRejectsRootSubstitutionAfterWatchRegistration(t *testing.T) {
	root := configuredRoot(t)
	operations := productionFileOperations()
	operations.afterWatcherAdd = func() {
		replaceConfiguredRoot(t, root)
	}

	reader, err := openReaderWithOperations(ReaderConfig{
		Root:               root,
		MaxPayloadBytes:    1024,
		SubscriptionBuffer: 2,
	}, operations)
	if reader != nil {
		_ = reader.Close()
		t.Fatal("Reader returned after configured root substitution")
	}
	if err == nil || !errors.Is(err, configcenter.ErrUnsafeState) {
		t.Fatalf("post-watch root substitution error = %v, want unsafe_state", err)
	}
}

func TestReaderGetDistinguishesMissingPresentEmptyAndCopiesBytes(t *testing.T) {
	root := t.TempDir()
	reader, err := OpenReader(ReaderConfig{Root: root, MaxPayloadBytes: 1024, SubscriptionBuffer: 2})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = reader.Close() }()
	key, err := configcenter.ParseKey("settings/alpha")
	if err != nil {
		t.Fatal(err)
	}

	missing, err := reader.Get(context.Background(), key)
	if err == nil || !errors.Is(err, configcenter.ErrMissing) || missing.Present() || missing.Revision().Scoped() {
		t.Fatalf("missing Get = present=%v revisionScoped=%v err=%v", missing.Present(), missing.Revision().Scoped(), err)
	}

	leaf, err := MapKey(key)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, leaf)
	if err := os.WriteFile(path, []byte{}, 0o600); err != nil {
		t.Fatal(err)
	}
	empty, err := reader.Get(context.Background(), key)
	if err != nil || !empty.Present() || len(empty.Content()) != 0 {
		t.Fatalf("present-empty Get = present=%v content=%v err=%v", empty.Present(), empty.Content(), err)
	}
	if err := os.WriteFile(path, []byte("value"), 0o600); err != nil {
		t.Fatal(err)
	}
	snapshot, err := reader.Get(context.Background(), key)
	if err != nil || !snapshot.Present() || string(snapshot.Content()) != "value" || snapshot.Revision().Scoped() {
		t.Fatalf("present Get = %#v err=%v", snapshot, err)
	}
	content := snapshot.Content()
	content[0] = 'X'
	again, err := reader.Get(context.Background(), key)
	if err != nil || string(again.Content()) != "value" {
		t.Fatalf("Get exposed retained bytes: %q, err=%v", again.Content(), err)
	}
}

func TestReaderGetMapsBoundsAndUnsafeState(t *testing.T) {
	root := t.TempDir()
	reader, err := OpenReader(ReaderConfig{Root: root, MaxPayloadBytes: 3, SubscriptionBuffer: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = reader.Close() }()
	key, err := configcenter.ParseKey("alpha")
	if err != nil {
		t.Fatal(err)
	}
	leaf, err := MapKey(key)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, leaf)
	if err := os.WriteFile(path, []byte("four"), 0o600); err != nil {
		t.Fatal(err)
	}
	if snapshot, err := reader.Get(context.Background(), key); err == nil || !errors.Is(err, configcenter.ErrPayloadTooLarge) || snapshot.Present() {
		t.Fatalf("oversize Get = present=%v err=%v", snapshot.Present(), err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "outside")
	if err := os.WriteFile(outside, []byte("outside"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, path); err != nil {
		if os.IsPermission(err) {
			t.Skip("test account cannot create a symlink")
		}
		t.Fatal(err)
	}
	if snapshot, err := reader.Get(context.Background(), key); err == nil || !errors.Is(err, configcenter.ErrUnsafeState) || snapshot.Present() || len(snapshot.Content()) != 0 {
		t.Fatalf("unsafe symlink Get = present=%v content=%q err=%v", snapshot.Present(), snapshot.Content(), err)
	}
}

func TestReaderGetRejectsNonRegularMappedLeafAndCancellation(t *testing.T) {
	root := t.TempDir()
	reader, err := OpenReader(ReaderConfig{Root: root, MaxPayloadBytes: 1024, SubscriptionBuffer: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = reader.Close() }()
	key, err := configcenter.ParseKey("alpha")
	if err != nil {
		t.Fatal(err)
	}
	leaf, err := MapKey(key)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, leaf)
	if err := os.Mkdir(path, 0o700); err != nil {
		t.Fatal(err)
	}
	if snapshot, err := reader.Get(context.Background(), key); err == nil || !errors.Is(err, configcenter.ErrUnsafeState) || snapshot.Present() {
		t.Fatalf("non-regular Get = present=%v err=%v", snapshot.Present(), err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := reader.Get(ctx, key); err == nil || !errors.Is(err, configcenter.ErrCanceled) {
		t.Fatalf("canceled Get = %v, want canceled", err)
	}
	var invalid configcenter.Key
	if _, err := reader.Get(context.Background(), invalid); err == nil || !errors.Is(err, configcenter.ErrInvalid) {
		t.Fatalf("invalid key Get = %v, want invalid", err)
	}
}

func TestReaderCloseIsIdempotentAndTerminal(t *testing.T) {
	root := t.TempDir()
	reader, err := OpenReader(ReaderConfig{Root: root, MaxPayloadBytes: 1024, SubscriptionBuffer: 1})
	if err != nil {
		t.Fatal(err)
	}
	if err := reader.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	if err := reader.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
	key, err := configcenter.ParseKey("alpha")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if _, err := reader.Get(ctx, key); err == nil || !errors.Is(err, configcenter.ErrReaderClosed) {
		t.Fatalf("Get after Close = %v, want reader_closed", err)
	}
}

func TestReadBoundedHandlesMaximumInt64WithoutOverflow(t *testing.T) {
	path := filepath.Join(t.TempDir(), "payload")
	if err := os.WriteFile(path, []byte("value"), 0o600); err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = file.Close() }()
	content, oversized, err := readBounded(file, math.MaxInt64)
	if err != nil || oversized || string(content) != "value" {
		t.Fatalf("readBounded(MaxInt64) = %q, oversized=%v, err=%v", content, oversized, err)
	}
}

func TestObserveDeliversOrderedUpdateAndDeleteFromMissing(t *testing.T) {
	root := t.TempDir()
	reader, err := OpenReader(ReaderConfig{Root: root, MaxPayloadBytes: 1024, SubscriptionBuffer: 4})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = reader.Close() }()
	key, err := configcenter.ParseKey("settings/alpha")
	if err != nil {
		t.Fatal(err)
	}
	observation, err := reader.Observe(context.Background(), key)
	if err != nil {
		t.Fatal(err)
	}
	if observation.Initial.Present() || !observation.Initial.Revision().Scoped() || observation.Initial.Revision().Order() != 0 {
		t.Fatalf("initial observation = %#v", observation.Initial)
	}
	leaf, err := MapKey(key)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, leaf)
	replaceFileAtomically(t, root, path, []byte("one"))
	update := nextFileEvent(t, observation.Subscription)
	if update.Kind() != configcenter.EventUpdate || string(update.Snapshot().Content()) != "one" || update.Snapshot().Revision().Order() != 1 {
		t.Fatalf("update event = %#v", update)
	}
	if err := configcenter.ValidateNextRevision(observation.Initial.Revision(), update.Snapshot().Revision()); err != nil {
		t.Fatalf("initial/update revision relationship: %v", err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	deletion := nextFileEvent(t, observation.Subscription)
	if deletion.Kind() != configcenter.EventDelete || deletion.Snapshot().Present() || deletion.Snapshot().Revision().Order() != 2 {
		t.Fatalf("delete event = %#v", deletion)
	}
	if err := configcenter.ValidateNextRevision(update.Snapshot().Revision(), deletion.Snapshot().Revision()); err != nil {
		t.Fatalf("update/delete revision relationship: %v", err)
	}
}

func TestObserveHandoffIncludesConcurrentReplacement(t *testing.T) {
	root := t.TempDir()
	reader, err := OpenReader(ReaderConfig{Root: root, MaxPayloadBytes: 1024, SubscriptionBuffer: 4})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = reader.Close() }()
	key, err := configcenter.ParseKey("alpha")
	if err != nil {
		t.Fatal(err)
	}
	leaf, err := MapKey(key)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, leaf)
	replaceFileAtomically(t, root, path, []byte("old"))

	type observeResult struct {
		observation configcenter.Observation
		err         error
	}
	result := make(chan observeResult, 1)
	go func() {
		observation, observeErr := reader.Observe(context.Background(), key)
		result <- observeResult{observation: observation, err: observeErr}
	}()
	replaceFileAtomically(t, root, path, []byte("new"))
	observed := <-result
	if observed.err != nil {
		t.Fatal(observed.err)
	}
	if got := string(observed.observation.Initial.Content()); got == "new" {
		return
	} else if got != "old" {
		t.Fatalf("initial content = %q, want old or new", got)
	}
	if event := nextFileEvent(t, observed.observation.Subscription); event.Kind() != configcenter.EventUpdate || string(event.Snapshot().Content()) != "new" {
		t.Fatalf("concurrent replacement was lost: %#v", event)
	}
}

func TestSubscriptionCancelCloseReaderCloseAndInterruptionStayDistinct(t *testing.T) {
	root := t.TempDir()
	reader, err := OpenReader(ReaderConfig{Root: root, MaxPayloadBytes: 1024, SubscriptionBuffer: 2})
	if err != nil {
		t.Fatal(err)
	}
	key, err := configcenter.ParseKey("alpha")
	if err != nil {
		t.Fatal(err)
	}
	observation, err := reader.Observe(context.Background(), key)
	if err != nil {
		t.Fatal(err)
	}
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := observation.Subscription.Next(canceled); err == nil || !errors.Is(err, configcenter.ErrCanceled) {
		t.Fatalf("canceled Next = %v, want canceled", err)
	}
	leaf, err := MapKey(key)
	if err != nil {
		t.Fatal(err)
	}
	replaceFileAtomically(t, root, filepath.Join(root, leaf), []byte("value"))
	if event := nextFileEvent(t, observation.Subscription); event.Kind() != configcenter.EventUpdate {
		t.Fatalf("canceled wait closed subscription: %#v", event)
	}
	if err := observation.Subscription.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := observation.Subscription.Next(context.Background()); err == nil || !errors.Is(err, configcenter.ErrSubscriptionClosed) {
		t.Fatalf("Next after subscription Close = %v", err)
	}

	second, err := reader.Observe(context.Background(), key)
	if err != nil {
		t.Fatal(err)
	}
	if err := reader.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := second.Subscription.Next(context.Background()); err == nil || !errors.Is(err, configcenter.ErrReaderClosed) {
		t.Fatalf("Next after reader Close = %v", err)
	}

	interruptedReader, err := OpenReader(ReaderConfig{Root: t.TempDir(), MaxPayloadBytes: 1024, SubscriptionBuffer: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = interruptedReader.Close() }()
	interrupted, err := interruptedReader.Observe(context.Background(), key)
	if err != nil {
		t.Fatal(err)
	}
	interruptedReader.interruptAll("")
	if _, err := interrupted.Subscription.Next(context.Background()); err == nil || !errors.Is(err, configcenter.ErrWatchInterrupted) {
		t.Fatalf("Next after interruption = %v", err)
	}
	if _, err := interruptedReader.Get(context.Background(), key); err == nil || !errors.Is(err, configcenter.ErrWatchInterrupted) {
		t.Fatalf("Get after interruption = %v", err)
	}
}

func TestWatchIgnoresUnrelatedChildrenAndRootRenameInterrupts(t *testing.T) {
	root := t.TempDir()
	reader, err := OpenReader(ReaderConfig{Root: root, MaxPayloadBytes: 1024, SubscriptionBuffer: 2})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = reader.Close() }()
	key, err := configcenter.ParseKey("alpha")
	if err != nil {
		t.Fatal(err)
	}
	observation, err := reader.Observe(context.Background(), key)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "unrelated.tmp"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	quiet, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	if _, err := observation.Subscription.Next(quiet); err == nil || !errors.Is(err, configcenter.ErrCanceled) {
		t.Fatalf("unrelated child altered subscription: %v", err)
	}
	moved := root + "-moved"
	if err := os.Rename(root, moved); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(moved) })
	if _, err := nextFileEventOrError(observation.Subscription); err == nil || !errors.Is(err, configcenter.ErrWatchInterrupted) {
		t.Fatalf("root rename Next = %v, want watch_interrupted", err)
	}
}

func TestEventStateFailureCarriesOnlySafeCauseKind(t *testing.T) {
	root := t.TempDir()
	reader, err := OpenReader(ReaderConfig{Root: root, MaxPayloadBytes: 1024, SubscriptionBuffer: 2})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = reader.Close() }()
	key, err := configcenter.ParseKey("alpha")
	if err != nil {
		t.Fatal(err)
	}
	leaf, err := MapKey(key)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, leaf)
	replaceFileAtomically(t, root, path, []byte("value"))
	observation, err := reader.Observe(context.Background(), key)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "outside")
	if err := os.WriteFile(outside, []byte("outside"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, path); err != nil {
		if os.IsPermission(err) {
			t.Skip("test account cannot create a symlink")
		}
		t.Fatal(err)
	}
	// Drive the same state-sampling path deterministically after constructing
	// the observed unsafe leaf; native notifications exercise it separately.
	reader.processKeyChange(key)
	_, terminalErr := nextFileEventOrError(observation.Subscription)
	if terminalErr == nil || !errors.Is(terminalErr, configcenter.ErrWatchInterrupted) {
		t.Fatalf("unsafe state event error = %v", terminalErr)
	}
	var typed *configcenter.Error
	if !errors.As(terminalErr, &typed) || typed.Details().CauseKind != configcenter.CodeUnsafeState {
		t.Fatalf("unsafe state cause details = %#v", typed)
	}
}

func TestDeliveryOverflowInterruptsRatherThanDroppingState(t *testing.T) {
	root := t.TempDir()
	reader, err := OpenReader(ReaderConfig{Root: root, MaxPayloadBytes: 1024, SubscriptionBuffer: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = reader.Close() }()
	key, err := configcenter.ParseKey("alpha")
	if err != nil {
		t.Fatal(err)
	}
	observation, err := reader.Observe(context.Background(), key)
	if err != nil {
		t.Fatal(err)
	}
	leaf, err := MapKey(key)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, leaf)
	replaceFileAtomically(t, root, path, []byte("one"))
	reader.processKeyChange(key)
	replaceFileAtomically(t, root, path, []byte("two"))
	reader.processKeyChange(key)
	if _, err := nextFileEventOrError(observation.Subscription); err == nil || !errors.Is(err, configcenter.ErrWatchInterrupted) {
		t.Fatalf("delivery overflow error = %v", err)
	}
}

func TestDuplicateCurrentStateNotificationIsSuppressedAndCounted(t *testing.T) {
	root := t.TempDir()
	reader, err := OpenReader(ReaderConfig{Root: root, MaxPayloadBytes: 1024, SubscriptionBuffer: 2})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = reader.Close() }()
	key, err := configcenter.ParseKey("alpha")
	if err != nil {
		t.Fatal(err)
	}
	observation, err := reader.Observe(context.Background(), key)
	if err != nil {
		t.Fatal(err)
	}
	leaf, err := MapKey(key)
	if err != nil {
		t.Fatal(err)
	}
	replaceFileAtomically(t, root, filepath.Join(root, leaf), []byte("one"))
	_ = nextFileEvent(t, observation.Subscription)
	before := observation.Subscription.Stats().SuppressedNotifications
	reader.processKeyChange(key)
	after := observation.Subscription.Stats().SuppressedNotifications
	if after != before+1 {
		t.Fatalf("suppressed count = %d, want %d", after, before+1)
	}
	quiet, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	if _, err := observation.Subscription.Next(quiet); err == nil || !errors.Is(err, configcenter.ErrCanceled) {
		t.Fatalf("duplicate notification created an event: %v", err)
	}
}

func replaceFileAtomically(t *testing.T, root, target string, content []byte) {
	t.Helper()
	temporary, err := os.CreateTemp(root, ".config-center-test-*")
	if err != nil {
		t.Fatal(err)
	}
	temporaryName := temporary.Name()
	if _, err := temporary.Write(content); err != nil {
		_ = temporary.Close()
		t.Fatal(err)
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		t.Fatal(err)
	}
	if err := temporary.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(temporaryName, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(temporaryName, target); err != nil {
		t.Fatal(err)
	}
}

func nextFileEvent(t *testing.T, subscription configcenter.Subscription) configcenter.ConfigurationEvent {
	t.Helper()
	event, err := nextFileEventOrError(subscription)
	if err != nil {
		t.Fatal(err)
	}
	return event
}

func nextFileEventOrError(subscription configcenter.Subscription) (configcenter.ConfigurationEvent, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return subscription.Next(ctx)
}
