package file

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"path/filepath"
	"sync"
	"sync/atomic"

	"github.com/fsnotify/fsnotify"

	configcenter "github.com/NeKiro-project/NeKiro/config_center"
)

type fileSubscription struct {
	reader   *Reader
	key      configcenter.Key
	events   chan configcenter.ConfigurationEvent
	done     chan struct{}
	terminal error

	terminalMu sync.Mutex
	suppressed atomic.Uint64

	// current and revision are protected by reader.mu.
	current  fileState
	revision configcenter.Revision
}

func newFileSubscription(reader *Reader, key configcenter.Key, current fileState, revision configcenter.Revision) *fileSubscription {
	return &fileSubscription{
		reader: reader,
		key:    key,
		events: make(chan configcenter.ConfigurationEvent, reader.subscriptionBuffer),
		done:   make(chan struct{}),
		current: fileState{
			present: current.present,
			content: bytes.Clone(current.content),
		},
		revision: revision,
	}
}

// Next returns one ordered state transition or a distinct lifecycle outcome.
// A canceled wait never closes the subscription.
func (subscription *fileSubscription) Next(ctx context.Context) (configcenter.ConfigurationEvent, error) {
	if err := contextError(ctx, subscription.key, configcenter.OperationNext); err != nil {
		return configcenter.ConfigurationEvent{}, err
	}
	select {
	case <-subscription.done:
		return configcenter.ConfigurationEvent{}, subscription.terminalError()
	default:
	}
	select {
	case <-subscription.done:
		return configcenter.ConfigurationEvent{}, subscription.terminalError()
	case event := <-subscription.events:
		return event, nil
	case <-ctx.Done():
		return configcenter.ConfigurationEvent{}, contextError(ctx, subscription.key, configcenter.OperationNext)
	}
}

// Close terminates this subscription only. It does not close the reader or
// recreate any source watch.
func (subscription *fileSubscription) Close() error {
	subscription.reader.closeSubscription(subscription)
	return nil
}

// Stats returns safe aggregate notification counters.
func (subscription *fileSubscription) Stats() configcenter.SubscriptionStats {
	return configcenter.SubscriptionStats{SuppressedNotifications: subscription.suppressed.Load()}
}

func (subscription *fileSubscription) terminate(err error) {
	subscription.terminalMu.Lock()
	defer subscription.terminalMu.Unlock()
	if subscription.terminal != nil {
		return
	}
	subscription.terminal = err
	close(subscription.done)
}

func (subscription *fileSubscription) terminalError() error {
	subscription.terminalMu.Lock()
	defer subscription.terminalMu.Unlock()
	if subscription.terminal == nil {
		panic("config center file: terminal subscription without terminal error")
	}
	return subscription.terminal
}

func (reader *Reader) watchLoop() {
	defer close(reader.watchDone)
	for {
		select {
		case event, ok := <-reader.watcher.Events():
			if !ok {
				reader.interruptAll("")
				return
			}
			reader.handleWatchEvent(event)
		case watcherErr, ok := <-reader.watcher.Errors():
			if !ok {
				reader.interruptAll("")
				return
			}
			reader.handleWatcherError(watcherErr)
		}
		if reader.stopped() {
			return
		}
	}
}

// handleWatcherError classifies the received provider error without replacing
// the existing watcher. Only the non-Darwin fsnotify source can report an
// ENOENT child-replacement error while its root watch remains valid. Native
// Darwin kqueue errors are terminal source faults.
func (reader *Reader) handleWatcherError(watcherErr error) {
	if reader.watcher.AllowsNotExistReconciliation() && errors.Is(watcherErr, fs.ErrNotExist) && !reader.rootPathUnsafeOrReplaced() {
		reader.processAllKeyChanges()
		return
	}
	reader.interruptAll("")
}

func (reader *Reader) stopped() bool {
	reader.mu.Lock()
	defer reader.mu.Unlock()
	return reader.closed || reader.interrupted
}

func (reader *Reader) handleWatchEvent(event fsnotify.Event) {
	name := filepath.Clean(event.Name)
	if name == reader.rootPath {
		if event.Has(fsnotify.Remove) || event.Has(fsnotify.Rename) || event.Has(fsnotify.Create) {
			reader.interruptAll("")
			return
		}
		if event.Has(fsnotify.Chmod) && reader.rootPathUnsafeOrReplaced() {
			reader.interruptAll("")
			return
		}
		// kqueue and Windows can report a direct-child change as a Write event
		// on the watched directory. This is an event-driven reconciliation of
		// the already registered keys, not a poll or source fallback.
		if event.Has(fsnotify.Write) {
			reader.processAllKeyChanges()
		}
		return
	}
	if filepath.Dir(name) != reader.rootPath {
		return
	}
	key, ok := mappedKeyFromEventName(filepath.Base(name))
	if !ok {
		return
	}
	reader.processKeyChange(key)
}

func (reader *Reader) processAllKeyChanges() {
	reader.mu.Lock()
	if reader.closed || reader.interrupted {
		reader.mu.Unlock()
		return
	}
	keys := make([]configcenter.Key, 0, len(reader.subscriptions))
	for key := range reader.subscriptions {
		keys = append(keys, key)
	}
	reader.mu.Unlock()
	for _, key := range keys {
		reader.processKeyChange(key)
		if reader.stopped() {
			return
		}
	}
}

func (reader *Reader) rootPathUnsafeOrReplaced() bool {
	return rootIdentityCode(reader.rootPath, reader.root, reader.rootIdentity, reader.operations) != ""
}

func (reader *Reader) processKeyChange(key configcenter.Key) {
	reader.mu.Lock()
	if reader.closed || reader.interrupted {
		reader.mu.Unlock()
		return
	}
	subscriptions := reader.subscriptions[key]
	if len(subscriptions) == 0 {
		reader.mu.Unlock()
		return
	}
	state, err := reader.readStateLocked(key, configcenter.OperationWatch)
	if err != nil {
		cause, _ := configcenter.CodeOf(err)
		for subscription := range subscriptions {
			subscription.terminate(watchInterruptedError(key, configcenter.OperationNext, watchCause(cause)))
		}
		delete(reader.subscriptions, key)
		reader.mu.Unlock()
		return
	}
	for subscription := range subscriptions {
		if sameFileState(subscription.current, state) {
			subscription.suppressed.Add(1)
			continue
		}
		nextRevision, advanceErr := configcenter.AdvanceRevision(subscription.revision)
		if advanceErr != nil {
			reader.mu.Unlock()
			reader.interruptAll("")
			return
		}
		snapshot, snapshotErr := snapshotForState(key, state, nextRevision)
		if snapshotErr != nil {
			reader.mu.Unlock()
			reader.interruptAll("")
			return
		}
		var event configcenter.ConfigurationEvent
		if state.present {
			event, snapshotErr = configcenter.NewUpdateEvent(snapshot)
		} else {
			event, snapshotErr = configcenter.NewDeleteEvent(snapshot)
		}
		if snapshotErr != nil || !subscription.enqueue(event) {
			reader.mu.Unlock()
			reader.interruptAll("")
			return
		}
		subscription.current = fileState{present: state.present, content: bytes.Clone(state.content)}
		subscription.revision = nextRevision
	}
	reader.mu.Unlock()
}

func (subscription *fileSubscription) enqueue(event configcenter.ConfigurationEvent) bool {
	select {
	case subscription.events <- event:
		return true
	default:
		return false
	}
}

func sameFileState(left, right fileState) bool {
	return left.present == right.present && (!left.present || bytes.Equal(left.content, right.content))
}

func watchCause(code configcenter.Code) configcenter.Code {
	switch code {
	case configcenter.CodeUnsafeState, configcenter.CodeUnauthorized, configcenter.CodeUnavailable, configcenter.CodePayloadTooLarge:
		return code
	default:
		return ""
	}
}

var _ configcenter.Subscription = (*fileSubscription)(nil)
