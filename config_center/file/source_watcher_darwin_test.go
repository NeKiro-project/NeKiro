//go:build darwin

package file

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	configcenter "github.com/NeKiro-project/NeKiro/config_center"
	"golang.org/x/sys/unix"
)

func TestDarwinKqueueRootWatchObservesExternalAtomicReplaceAndDelete(t *testing.T) {
	root := t.TempDir()
	reader, err := OpenReader(ReaderConfig{Root: root, MaxPayloadBytes: 1024, SubscriptionBuffer: 2})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = reader.Close() }()

	watcher, ok := reader.watcher.(*kqueueRootWatcher)
	if !ok {
		t.Fatalf("Darwin source = %T, want *kqueueRootWatcher", reader.watcher)
	}
	if watcher.AllowsNotExistReconciliation() {
		t.Fatal("Darwin kqueue source permits non-Darwin ENOENT reconciliation")
	}
	if !os.SameFile(watcher.Identity(), reader.rootIdentity) {
		t.Fatal("Darwin root source identity does not match pinned root")
	}

	key, err := configcenter.ParseKey("darwin/root-watch")
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

	replaceFileAtomically(t, root, path, []byte("external"))
	update := nextFileEvent(t, observation.Subscription)
	if update.Kind() != configcenter.EventUpdate || string(update.Snapshot().Content()) != "external" {
		t.Fatalf("external replacement event = %#v", update)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	deletion := nextFileEvent(t, observation.Subscription)
	if deletion.Kind() != configcenter.EventDelete || deletion.Snapshot().Present() {
		t.Fatalf("external deletion event = %#v", deletion)
	}
}

func TestDarwinKqueueRootWatchCloseWaitsForWorker(t *testing.T) {
	watcher, err := openSourceWatcher(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	native, ok := watcher.(*kqueueRootWatcher)
	if !ok {
		_ = watcher.Close()
		t.Fatalf("Darwin source = %T, want *kqueueRootWatcher", watcher)
	}
	if err := native.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-native.done:
	default:
		t.Fatal("native kqueue worker remains live after Close")
	}
}

func TestCompleteKeventContinuesOnlyEINTR(t *testing.T) {
	calls := 0
	call := func(int, []unix.Kevent_t, []unix.Kevent_t, *unix.Timespec) (int, error) {
		calls++
		if calls == 1 {
			return 0, unix.EINTR
		}
		return 1, nil
	}

	count, err := completeKevent(call, 1, nil, nil, nil)
	if err != nil || count != 1 || calls != 2 {
		t.Fatalf("completeKevent = (%d, %v) after %d calls, want (1, nil) after 2", count, err, calls)
	}
}

func TestCompleteKeventReturnsSourceError(t *testing.T) {
	want := errors.New("kqueue source failure")
	calls := 0
	call := func(int, []unix.Kevent_t, []unix.Kevent_t, *unix.Timespec) (int, error) {
		calls++
		return 0, want
	}

	_, err := completeKevent(call, 1, nil, nil, nil)
	if !errors.Is(err, want) || calls != 1 {
		t.Fatalf("completeKevent error = %v after %d calls, want source error after one call", err, calls)
	}
}
