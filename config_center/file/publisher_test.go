package file

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	configcenter "github.com/NeKiro-project/NeKiro/config_center"
)

func TestPublisherCopiesInputUsesExplicitModeAndTemporaryMappingIsolation(t *testing.T) {
	root := t.TempDir()
	publisher, err := OpenPublisher(PublisherConfig{Root: root, MaxPayloadBytes: 1024, FileMode: testPublisherMode()})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = publisher.Close() }()
	key, err := configcenter.ParseKey("alpha")
	if err != nil {
		t.Fatal(err)
	}
	payload := []byte("value")
	if err := publisher.Publish(context.Background(), key, payload); err != nil {
		t.Fatal(err)
	}
	payload[0] = 'X'
	leaf, err := MapKey(key)
	if err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join(root, leaf))
	if err != nil || string(content) != "value" {
		t.Fatalf("published content = %q, %v", content, err)
	}
	info, err := os.Lstat(filepath.Join(root, leaf))
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != testPublisherMode() {
		t.Fatalf("published mode = %#o, want %#o", got, testPublisherMode())
	}
	temporary, err := publisher.temporaryLeaf()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := mappedKeyFromEventName(temporary); ok {
		t.Fatalf("publisher temporary leaf was interpreted as a config key: %q", temporary)
	}
}

func TestPublisherAtomicReplacementDeliversCompleteReaderState(t *testing.T) {
	root := t.TempDir()
	reader, err := OpenReader(ReaderConfig{Root: root, MaxPayloadBytes: 1024, SubscriptionBuffer: 4})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = reader.Close() }()
	publisher, err := OpenPublisher(PublisherConfig{Root: root, MaxPayloadBytes: 1024, FileMode: testWritablePublisherMode()})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = publisher.Close() }()
	key, err := configcenter.ParseKey("alpha")
	if err != nil {
		t.Fatal(err)
	}
	if err := publisher.Publish(context.Background(), key, []byte("complete-old")); err != nil {
		t.Fatal(err)
	}
	observation, err := reader.Observe(context.Background(), key)
	if err != nil || string(observation.Initial.Content()) != "complete-old" {
		t.Fatalf("initial state = %#v, %v", observation.Initial, err)
	}
	if err := publisher.Publish(context.Background(), key, []byte("complete-new")); err != nil {
		t.Fatal(err)
	}
	event := nextFileEvent(t, observation.Subscription)
	if event.Kind() != configcenter.EventUpdate || string(event.Snapshot().Content()) != "complete-new" {
		t.Fatalf("atomic replacement event = %#v", event)
	}
	current, err := reader.Get(context.Background(), key)
	if err != nil || string(current.Content()) != "complete-new" {
		t.Fatalf("current state after replacement = %#v, %v", current, err)
	}
}

func TestPublisherMapsMissingUnsafeBoundsCancellationAndClose(t *testing.T) {
	root := t.TempDir()
	publisher, err := OpenPublisher(PublisherConfig{Root: root, MaxPayloadBytes: 3, FileMode: testWritablePublisherMode()})
	if err != nil {
		t.Fatal(err)
	}
	key, err := configcenter.ParseKey("alpha")
	if err != nil {
		t.Fatal(err)
	}
	if err := publisher.Delete(context.Background(), key); err == nil || !errors.Is(err, configcenter.ErrMissing) {
		t.Fatalf("Delete missing = %v", err)
	}
	if err := publisher.Publish(context.Background(), key, []byte("four")); err == nil || !errors.Is(err, configcenter.ErrPayloadTooLarge) {
		t.Fatalf("oversize Publish = %v", err)
	}
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if err := publisher.Publish(canceled, key, []byte("one")); err == nil || !errors.Is(err, configcenter.ErrCanceled) {
		t.Fatalf("canceled Publish = %v", err)
	}

	leaf, err := MapKey(key)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, leaf)
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
	for _, operation := range []func() error{
		func() error { return publisher.Publish(context.Background(), key, []byte("one")) },
		func() error { return publisher.Delete(context.Background(), key) },
	} {
		if err := operation(); err == nil || !errors.Is(err, configcenter.ErrUnsafeState) {
			t.Fatalf("unsafe target operation = %v", err)
		}
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(path, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := publisher.Publish(context.Background(), key, []byte("one")); err == nil || !errors.Is(err, configcenter.ErrUnsafeState) {
		t.Fatalf("non-regular target Publish = %v", err)
	}
	if err := publisher.Close(); err != nil {
		t.Fatal(err)
	}
	if err := publisher.Close(); err != nil {
		t.Fatal(err)
	}
	if err := publisher.Publish(context.Background(), key, []byte("one")); err == nil || !errors.Is(err, configcenter.ErrPublisherClosed) {
		t.Fatalf("Publish after Close = %v", err)
	}
	if err := publisher.Delete(context.Background(), key); err == nil || !errors.Is(err, configcenter.ErrPublisherClosed) {
		t.Fatalf("Delete after Close = %v", err)
	}
}

func TestPublisherRejectsRootLossAndUnsafeSubstitutionAtOperationStart(t *testing.T) {
	key, err := configcenter.ParseKey("alpha")
	if err != nil {
		t.Fatal(err)
	}

	t.Run("missing root is unavailable", func(t *testing.T) {
		skipWindowsRootPathMutation(t)
		root := configuredRoot(t)
		publisher, err := OpenPublisher(PublisherConfig{Root: root, MaxPayloadBytes: 1024, FileMode: testWritablePublisherMode()})
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = publisher.Close() }()
		if err := os.Rename(root, root+"-moved"); err != nil {
			t.Fatal(err)
		}
		for _, operation := range []func() error{
			func() error { return publisher.Publish(context.Background(), key, []byte("value")) },
			func() error { return publisher.Delete(context.Background(), key) },
		} {
			if err := operation(); err == nil || !errors.Is(err, configcenter.ErrUnavailable) {
				t.Fatalf("operation after root loss = %v, want unavailable", err)
			}
		}
	})

	t.Run("replacement root is unsafe", func(t *testing.T) {
		skipWindowsRootPathMutation(t)
		root := configuredRoot(t)
		publisher, err := OpenPublisher(PublisherConfig{Root: root, MaxPayloadBytes: 1024, FileMode: testWritablePublisherMode()})
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = publisher.Close() }()
		replaceConfiguredRoot(t, root)
		for _, operation := range []func() error{
			func() error { return publisher.Publish(context.Background(), key, []byte("value")) },
			func() error { return publisher.Delete(context.Background(), key) },
		} {
			if err := operation(); err == nil || !errors.Is(err, configcenter.ErrUnsafeState) {
				t.Fatalf("operation after root replacement = %v, want unsafe_state", err)
			}
		}
	})

	t.Run("non-directory root is unsafe", func(t *testing.T) {
		skipWindowsRootPathMutation(t)
		root := configuredRoot(t)
		publisher, err := OpenPublisher(PublisherConfig{Root: root, MaxPayloadBytes: 1024, FileMode: testWritablePublisherMode()})
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = publisher.Close() }()
		replaceConfiguredRootWithFile(t, root)
		for _, operation := range []func() error{
			func() error { return publisher.Publish(context.Background(), key, []byte("value")) },
			func() error { return publisher.Delete(context.Background(), key) },
		} {
			if err := operation(); err == nil || !errors.Is(err, configcenter.ErrUnsafeState) {
				t.Fatalf("operation after non-directory substitution = %v, want unsafe_state", err)
			}
		}
	})

	t.Run("symlink root is unsafe", func(t *testing.T) {
		skipWindowsRootPathMutation(t)
		root := configuredRoot(t)
		publisher, err := OpenPublisher(PublisherConfig{Root: root, MaxPayloadBytes: 1024, FileMode: testWritablePublisherMode()})
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = publisher.Close() }()
		replaceConfiguredRootWithSymlink(t, root)
		for _, operation := range []func() error{
			func() error { return publisher.Publish(context.Background(), key, []byte("value")) },
			func() error { return publisher.Delete(context.Background(), key) },
		} {
			if err := operation(); err == nil || !errors.Is(err, configcenter.ErrUnsafeState) {
				t.Fatalf("operation after root symlink substitution = %v, want unsafe_state", err)
			}
		}
	})

	t.Run("permission is unauthorized", func(t *testing.T) {
		root := configuredRoot(t)
		denyPathStat := false
		operations := productionFileOperations()
		platformLstat := operations.lstat
		operations.lstat = func(path string) (fs.FileInfo, error) {
			if denyPathStat {
				return nil, fs.ErrPermission
			}
			return platformLstat(path)
		}
		publisher, err := openPublisherWithOperations(PublisherConfig{Root: root, MaxPayloadBytes: 1024, FileMode: testWritablePublisherMode()}, operations)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = publisher.Close() }()
		denyPathStat = true
		for _, operation := range []func() error{
			func() error { return publisher.Publish(context.Background(), key, []byte("value")) },
			func() error { return publisher.Delete(context.Background(), key) },
		} {
			if err := operation(); err == nil || !errors.Is(err, configcenter.ErrUnauthorized) {
				t.Fatalf("operation after root permission failure = %v, want unauthorized", err)
			}
		}
	})
}

func TestPublisherRejectsRootReplacementBeforeFinalCommit(t *testing.T) {
	key, err := configcenter.ParseKey("alpha")
	if err != nil {
		t.Fatal(err)
	}
	leaf, err := MapKey(key)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("publish", func(t *testing.T) {
		skipWindowsRootPathMutation(t)
		root := configuredRoot(t)
		operations := productionFileOperations()
		operations.beforePublisherCommit = func() { replaceConfiguredRoot(t, root) }
		publisher, err := openPublisherWithOperations(PublisherConfig{Root: root, MaxPayloadBytes: 1024, FileMode: testWritablePublisherMode()}, operations)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = publisher.Close() }()
		if err := publisher.Publish(context.Background(), key, []byte("value")); err == nil || !errors.Is(err, configcenter.ErrUnsafeState) {
			t.Fatalf("Publish after pre-rename root replacement = %v, want unsafe_state", err)
		}
		assertPathAbsent(t, filepath.Join(root, leaf))
		assertPathAbsent(t, filepath.Join(filepath.Dir(root), "previous", leaf))
	})

	t.Run("delete", func(t *testing.T) {
		skipWindowsRootPathMutation(t)
		root := configuredRoot(t)
		if err := os.WriteFile(filepath.Join(root, leaf), []byte("value"), 0o600); err != nil {
			t.Fatal(err)
		}
		operations := productionFileOperations()
		operations.beforePublisherCommit = func() { replaceConfiguredRoot(t, root) }
		publisher, err := openPublisherWithOperations(PublisherConfig{Root: root, MaxPayloadBytes: 1024, FileMode: testWritablePublisherMode()}, operations)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = publisher.Close() }()
		if err := publisher.Delete(context.Background(), key); err == nil || !errors.Is(err, configcenter.ErrUnsafeState) {
			t.Fatalf("Delete after pre-remove root replacement = %v, want unsafe_state", err)
		}
		if _, err := os.Lstat(filepath.Join(filepath.Dir(root), "previous", leaf)); err != nil {
			t.Fatalf("Delete changed the old pinned target before failing: %v", err)
		}
	})
}

func TestPublisherReturnsTypedFailureWhenRootChangesAfterCommit(t *testing.T) {
	key, err := configcenter.ParseKey("alpha")
	if err != nil {
		t.Fatal(err)
	}
	leaf, err := MapKey(key)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("publish", func(t *testing.T) {
		skipWindowsRootPathMutation(t)
		root := configuredRoot(t)
		operations := productionFileOperations()
		operations.beforePublisherSuccess = func() { replaceConfiguredRoot(t, root) }
		publisher, err := openPublisherWithOperations(PublisherConfig{Root: root, MaxPayloadBytes: 1024, FileMode: testWritablePublisherMode()}, operations)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = publisher.Close() }()
		if err := publisher.Publish(context.Background(), key, []byte("value")); err == nil || !errors.Is(err, configcenter.ErrUnsafeState) {
			t.Fatalf("Publish after committed old-root write = %v, want unsafe_state", err)
		}
		assertPathAbsent(t, filepath.Join(root, leaf))
		content, err := os.ReadFile(filepath.Join(filepath.Dir(root), "previous", leaf))
		if err != nil || string(content) != "value" {
			t.Fatalf("old pinned publish = %q, %v", content, err)
		}
	})

	t.Run("delete", func(t *testing.T) {
		skipWindowsRootPathMutation(t)
		root := configuredRoot(t)
		if err := os.WriteFile(filepath.Join(root, leaf), []byte("value"), 0o600); err != nil {
			t.Fatal(err)
		}
		operations := productionFileOperations()
		operations.beforePublisherSuccess = func() { replaceConfiguredRoot(t, root) }
		publisher, err := openPublisherWithOperations(PublisherConfig{Root: root, MaxPayloadBytes: 1024, FileMode: testWritablePublisherMode()}, operations)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = publisher.Close() }()
		if err := publisher.Delete(context.Background(), key); err == nil || !errors.Is(err, configcenter.ErrUnsafeState) {
			t.Fatalf("Delete after committed old-root removal = %v, want unsafe_state", err)
		}
		assertPathAbsent(t, filepath.Join(filepath.Dir(root), "previous", leaf))
	})
}

func replaceConfiguredRootWithSymlink(t *testing.T, root string) {
	t.Helper()
	previous := filepath.Join(filepath.Dir(root), "previous")
	if err := os.Rename(root, previous); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(previous, root); err != nil {
		if os.IsPermission(err) {
			t.Skip("test account cannot create a symlink")
		}
		t.Fatal(err)
	}
}

func skipWindowsRootPathMutation(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("Windows directory handles prevent this pathname mutation")
	}
}

func replaceConfiguredRootWithFile(t *testing.T, root string) {
	t.Helper()
	previous := filepath.Join(filepath.Dir(root), "previous")
	if err := os.Rename(root, previous); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(root, []byte("not-a-directory"), 0o600); err != nil {
		t.Fatal(err)
	}
}

func assertPathAbsent(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Lstat(path); err == nil || !os.IsNotExist(err) {
		t.Fatalf("path %q state error = %v, want not exist", path, err)
	}
}

var _ configcenter.ConfigurationPublisher = (*Publisher)(nil)
