package file

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	configcenter "github.com/NeKiro-project/NeKiro/config_center"
)

func TestHandleWatcherErrorNotExistWithIntactRootReconcilesAndContinues(t *testing.T) {
	root, reader := newManualWatcherReader(t)
	defer func() { _ = reader.Close() }()
	key := watcherErrorTestKey(t)

	observation, err := reader.Observe(context.Background(), key)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS == "darwin" {
		reader.handleWatcherError(&fs.PathError{Op: "watch", Err: fs.ErrNotExist})
		assertWatcherErrorInterrupted(t, reader, key, observation.Subscription)
		return
	}
	leaf, err := MapKey(key)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, leaf), []byte("reconciled"), 0o600); err != nil {
		t.Fatal(err)
	}

	reader.handleWatcherError(&fs.PathError{Op: "watch", Err: fs.ErrNotExist})

	event := nextFileEvent(t, observation.Subscription)
	if event.Kind() != configcenter.EventUpdate || string(event.Snapshot().Content()) != "reconciled" {
		t.Fatalf("reconciled event = %#v", event)
	}
	if watchList := reader.watcher.WatchList(); len(watchList) != 1 || filepath.Clean(watchList[0]) != filepath.Clean(root) {
		t.Fatalf("watch list after reconciliation = %#v, want existing root %q", watchList, root)
	}
	if snapshot, err := reader.Get(context.Background(), key); err != nil || string(snapshot.Content()) != "reconciled" {
		t.Fatalf("Get after intact-root ENOENT = %#v, %v", snapshot, err)
	}
}

func TestHandleWatcherErrorNotExistWithLostOrReplacedRootInterrupts(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(*testing.T, string)
	}{
		{
			name: "lost",
			mutate: func(t *testing.T, root string) {
				t.Helper()
				if err := os.RemoveAll(root); err != nil {
					t.Fatal(err)
				}
			},
		},
		{name: "replaced", mutate: replaceConfiguredRoot},
	} {
		t.Run(test.name, func(t *testing.T) {
			root, reader := newManualWatcherReader(t)
			defer func() { _ = reader.Close() }()
			key := watcherErrorTestKey(t)
			observation, err := reader.Observe(context.Background(), key)
			if err != nil {
				t.Fatal(err)
			}

			test.mutate(t, root)
			reader.handleWatcherError(fs.ErrNotExist)
			assertWatcherErrorInterrupted(t, reader, key, observation.Subscription)
		})
	}
}

func TestHandleWatcherErrorGenericErrorInterrupts(t *testing.T) {
	_, reader := newManualWatcherReader(t)
	defer func() { _ = reader.Close() }()
	key := watcherErrorTestKey(t)
	observation, err := reader.Observe(context.Background(), key)
	if err != nil {
		t.Fatal(err)
	}

	reader.handleWatcherError(errors.New("watcher failed"))
	assertWatcherErrorInterrupted(t, reader, key, observation.Subscription)
}

func newManualWatcherReader(t *testing.T) (string, *Reader) {
	t.Helper()
	rootPath := configuredRoot(t)
	root, identity, err := openPinnedRoot(rootPath, configcenter.OperationObserve)
	if err != nil {
		t.Fatal(err)
	}
	watcher, err := openSourceWatcher(rootPath)
	if err != nil {
		_ = root.Close()
		t.Fatal(err)
	}
	return rootPath, &Reader{
		root:               root,
		rootPath:           rootPath,
		rootIdentity:       identity,
		operations:         productionFileOperations(),
		watcher:            watcher,
		maxPayloadBytes:    1024,
		subscriptionBuffer: 4,
		subscriptions:      make(map[configcenter.Key]map[*fileSubscription]struct{}),
		closeDone:          make(chan struct{}),
		watchDone:          make(chan struct{}),
	}
}

func watcherErrorTestKey(t *testing.T) configcenter.Key {
	t.Helper()
	key, err := configcenter.ParseKey("watcher/error")
	if err != nil {
		t.Fatal(err)
	}
	return key
}

func assertWatcherErrorInterrupted(t *testing.T, reader *Reader, key configcenter.Key, subscription configcenter.Subscription) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if _, err := subscription.Next(ctx); err == nil || !errors.Is(err, configcenter.ErrWatchInterrupted) {
		t.Fatalf("Next after watcher error = %v, want watch_interrupted", err)
	}
	if _, err := reader.Get(context.Background(), key); err == nil || !errors.Is(err, configcenter.ErrWatchInterrupted) {
		t.Fatalf("Get after watcher error = %v, want watch_interrupted", err)
	}
}
