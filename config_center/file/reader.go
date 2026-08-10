package file

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"runtime"
	"sync"

	configcenter "github.com/NeKiro-project/NeKiro/config_center"
)

// Reader is a read-only File-backed DynamicConfiguration capability.
type Reader struct {
	root               *os.Root
	rootPath           string
	rootIdentity       fs.FileInfo
	operations         fileOperations
	watcher            sourceWatcher
	maxPayloadBytes    int64
	subscriptionBuffer int

	mu            sync.Mutex
	closed        bool
	interrupted   bool
	subscriptions map[configcenter.Key]map[*fileSubscription]struct{}
	closeDone     chan struct{}
	closeErr      error
	watchDone     chan struct{}
	resourceOnce  sync.Once
	resourceErr   error
}

// OpenReader opens one explicit root and establishes exactly one non-recursive
// watch on it. The root descriptor is retained for every mapped read.
func OpenReader(config ReaderConfig) (*Reader, error) {
	return openReaderWithOperations(config, productionFileOperations())
}

func openReaderWithOperations(config ReaderConfig, operations fileOperations) (*Reader, error) {
	if err := validateReaderConfig(config); err != nil {
		return nil, err
	}
	root, rootIdentity, err := openPinnedRootWithOperations(config.Root, configcenter.OperationObserve, operations)
	if err != nil {
		return nil, err
	}
	watcher, err := openSourceWatcher(config.Root)
	if err != nil {
		closeRootIgnoringError(root)
		return nil, mapLeafFilesystemError(configcenter.Key{}, configcenter.OperationWatch, err)
	}
	invokeFileHook(operations.afterWatcherAdd)
	if code := rootIdentityCode(config.Root, root, rootIdentity, operations); code != "" || !watcherIdentityMatches(watcher, rootIdentity) {
		closeWatcherIgnoringError(watcher)
		closeRootIgnoringError(root)
		if code == "" {
			code = configcenter.CodeUnsafeState
		}
		return nil, fileError(code, configcenter.Key{}, configcenter.OperationWatch)
	}
	reader := &Reader{
		root:               root,
		rootPath:           config.Root,
		rootIdentity:       rootIdentity,
		operations:         operations,
		watcher:            watcher,
		maxPayloadBytes:    config.MaxPayloadBytes,
		subscriptionBuffer: config.SubscriptionBuffer,
		subscriptions:      make(map[configcenter.Key]map[*fileSubscription]struct{}),
		closeDone:          make(chan struct{}),
		watchDone:          make(chan struct{}),
	}
	go reader.watchLoop()
	return reader, nil
}

// Get reads the current File state through the pinned root. Its successful and
// missing snapshots always have an explicitly unscoped revision.
func (reader *Reader) Get(ctx context.Context, key configcenter.Key) (configcenter.Snapshot, error) {
	if err := contextError(ctx, key, configcenter.OperationGet); err != nil {
		return configcenter.Snapshot{}, err
	}
	if !key.Valid() {
		return configcenter.Snapshot{}, fileError(configcenter.CodeInvalid, key, configcenter.OperationGet)
	}

	reader.mu.Lock()
	defer reader.mu.Unlock()
	if reader.closed {
		return configcenter.Snapshot{}, fileError(configcenter.CodeReaderClosed, key, configcenter.OperationGet)
	}
	if reader.interrupted {
		return configcenter.Snapshot{}, watchInterruptedError(key, configcenter.OperationGet, "")
	}
	state, err := reader.readStateLocked(key, configcenter.OperationGet)
	if err != nil {
		return configcenter.Snapshot{}, err
	}
	if err := contextError(ctx, key, configcenter.OperationGet); err != nil {
		return configcenter.Snapshot{}, err
	}
	if !state.present {
		snapshot, err := configcenter.NewMissingSnapshot(key, configcenter.UnscopedRevision())
		if err != nil {
			return configcenter.Snapshot{}, err
		}
		return snapshot, fileError(configcenter.CodeMissing, key, configcenter.OperationGet)
	}
	snapshot, err := configcenter.NewPresentSnapshot(key, state.content, configcenter.UnscopedRevision())
	if err != nil {
		return configcenter.Snapshot{}, err
	}
	return snapshot, nil
}

// Close releases the root watch and pinned root. It is idempotent. The first
// physical close may report an unavailable outcome, but the reader remains
// closed regardless and does not reopen either resource.
func (reader *Reader) Close() error {
	reader.mu.Lock()
	if reader.closed {
		done := reader.closeDone
		reader.mu.Unlock()
		<-done
		reader.mu.Lock()
		err := reader.closeErr
		reader.mu.Unlock()
		return err
	}
	reader.closed = true
	reader.terminateAllLocked(func(key configcenter.Key) error {
		return fileError(configcenter.CodeReaderClosed, key, configcenter.OperationNext)
	})
	reader.mu.Unlock()

	closeErr := reader.closeResources()

	reader.mu.Lock()
	reader.closeErr = closeErr
	close(reader.closeDone)
	reader.mu.Unlock()
	return closeErr
}

// Observe atomically samples one key and registers its continuing subscription.
// The watcher uses the same reader lock, so a concurrent source transition is
// represented by the returned initial state or a later event, never lost in a
// Get-then-watch gap.
func (reader *Reader) Observe(ctx context.Context, key configcenter.Key) (configcenter.Observation, error) {
	if err := contextError(ctx, key, configcenter.OperationObserve); err != nil {
		return configcenter.Observation{}, err
	}
	if !key.Valid() {
		return configcenter.Observation{}, fileError(configcenter.CodeInvalid, key, configcenter.OperationObserve)
	}

	reader.mu.Lock()
	if reader.closed {
		reader.mu.Unlock()
		return configcenter.Observation{}, fileError(configcenter.CodeReaderClosed, key, configcenter.OperationObserve)
	}
	if reader.interrupted {
		reader.mu.Unlock()
		return configcenter.Observation{}, watchInterruptedError(key, configcenter.OperationObserve, "")
	}
	state, err := reader.readStateLocked(key, configcenter.OperationObserve)
	if err != nil {
		reader.mu.Unlock()
		return configcenter.Observation{}, err
	}
	if err := contextError(ctx, key, configcenter.OperationObserve); err != nil {
		reader.mu.Unlock()
		return configcenter.Observation{}, err
	}

	revision := configcenter.NewObservationRevision()
	initial, err := snapshotForState(key, state, revision)
	if err != nil {
		reader.mu.Unlock()
		return configcenter.Observation{}, err
	}
	subscription := newFileSubscription(reader, key, state, revision)
	if reader.subscriptions[key] == nil {
		reader.subscriptions[key] = make(map[*fileSubscription]struct{})
	}
	reader.subscriptions[key][subscription] = struct{}{}
	reader.mu.Unlock()

	if err := contextError(ctx, key, configcenter.OperationObserve); err != nil {
		_ = subscription.Close()
		return configcenter.Observation{}, err
	}
	return configcenter.Observation{Initial: initial, Subscription: subscription}, nil
}

type fileState struct {
	present bool
	content []byte
}

func (reader *Reader) readStateLocked(key configcenter.Key, operation configcenter.Operation) (fileState, error) {
	leaf, err := MapKey(key)
	if err != nil {
		return fileState{}, fileError(configcenter.CodeInvalid, key, operation)
	}
	info, err := reader.operations.rootLeafLstat(reader.root, leaf)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) || os.IsNotExist(err) {
			return fileState{}, nil
		}
		if runtime.GOOS == "windows" {
			// Windows can report a sharing violation while os.Root inspects a
			// reparse-point leaf. Classify the pathname object without following
			// it; successful reads still use only the confined root handle.
			missing, mappedErr := reader.mapLeafAccessFailure(key, operation, leaf, err)
			if missing {
				return fileState{}, nil
			}
			return fileState{}, mappedErr
		}
		return fileState{}, mapLeafFilesystemError(key, operation, err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return fileState{}, fileError(configcenter.CodeUnsafeState, key, operation)
	}
	if info.Size() > reader.maxPayloadBytes {
		return fileState{}, fileError(configcenter.CodePayloadTooLarge, key, operation)
	}

	opened, err := reader.operations.rootLeafOpen(reader.root, leaf)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) || os.IsNotExist(err) {
			return fileState{}, nil
		}
		if runtime.GOOS == "windows" {
			missing, mappedErr := reader.mapLeafAccessFailure(key, operation, leaf, err)
			if missing {
				return fileState{}, nil
			}
			return fileState{}, mappedErr
		}
		return fileState{}, mapLeafFilesystemError(key, operation, err)
	}
	content, oversized, readErr := readBounded(opened, reader.maxPayloadBytes)
	closeErr := opened.Close()
	if readErr != nil {
		return fileState{}, mapLeafFilesystemError(key, operation, readErr)
	}
	if closeErr != nil {
		return fileState{}, mapLeafFilesystemError(key, operation, closeErr)
	}
	if oversized {
		return fileState{}, fileError(configcenter.CodePayloadTooLarge, key, operation)
	}
	return fileState{present: true, content: bytes.Clone(content)}, nil
}

type leafFailureState uint8

const (
	leafFailureUnknown leafFailureState = iota
	leafFailureMissing
	leafFailureUnsafe
)

func (reader *Reader) mapLeafAccessFailure(key configcenter.Key, operation configcenter.Operation, leaf string, err error) (bool, error) {
	if code := rootIdentityCode(reader.rootPath, reader.root, reader.rootIdentity, reader.operations); code != "" {
		return false, fileError(code, key, operation)
	}
	return mapLeafAccessFailure(key, operation, reader.rootPath, leaf, err)
}

func mapLeafAccessFailure(key configcenter.Key, operation configcenter.Operation, rootPath, leaf string, err error) (bool, error) {
	switch leafStateAfterAccessFailure(rootPath, leaf) {
	case leafFailureMissing:
		return true, nil
	case leafFailureUnsafe:
		return false, fileError(configcenter.CodeUnsafeState, key, operation)
	default:
		return false, mapLeafFilesystemError(key, operation, err)
	}
}

func readBounded(file *os.File, maximum int64) ([]byte, bool, error) {
	// Read at most the caller-authorized amount first. A separate one-byte
	// probe detects overflow without forming maximum+1, which would overflow
	// when a caller explicitly selects math.MaxInt64.
	content, err := io.ReadAll(io.LimitReader(file, maximum))
	if err != nil {
		return nil, false, err
	}
	if int64(len(content)) != maximum {
		return content, false, nil
	}
	var extra [1]byte
	count, err := file.Read(extra[:])
	if count > 0 {
		return nil, true, nil
	}
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, false, err
	}
	return content, false, nil
}

func mapLeafFilesystemError(key configcenter.Key, operation configcenter.Operation, err error) error {
	if errors.Is(err, fs.ErrPermission) || os.IsPermission(err) {
		return fileError(configcenter.CodeUnauthorized, key, operation)
	}
	return fileError(configcenter.CodeUnavailable, key, operation)
}

func contextError(ctx context.Context, key configcenter.Key, operation configcenter.Operation) error {
	if err := ctx.Err(); err != nil {
		return configcenter.NewCanceledError(configcenter.ErrorDetails{
			Provider:  fileProvider,
			Key:       key,
			Operation: operation,
		}, err)
	}
	return nil
}

func (reader *Reader) closeResources() error {
	reader.resourceOnce.Do(func() {
		watcherErr := reader.watcher.Close()
		rootErr := reader.root.Close()
		if watcherErr != nil && !errors.Is(watcherErr, fs.ErrClosed) {
			reader.resourceErr = fileError(configcenter.CodeUnavailable, configcenter.Key{}, configcenter.OperationClose)
			return
		}
		if rootErr != nil {
			reader.resourceErr = fileError(configcenter.CodeUnavailable, configcenter.Key{}, configcenter.OperationClose)
		}
	})
	return reader.resourceErr
}

func (reader *Reader) terminateAllLocked(errorFor func(configcenter.Key) error) {
	for key, subscriptions := range reader.subscriptions {
		for subscription := range subscriptions {
			subscription.terminate(errorFor(key))
		}
	}
	reader.subscriptions = make(map[configcenter.Key]map[*fileSubscription]struct{})
}

func (reader *Reader) interruptAll(cause configcenter.Code) {
	reader.mu.Lock()
	if reader.closed || reader.interrupted {
		reader.mu.Unlock()
		return
	}
	reader.interrupted = true
	reader.terminateAllLocked(func(key configcenter.Key) error {
		return watchInterruptedError(key, configcenter.OperationNext, cause)
	})
	reader.mu.Unlock()
	_ = reader.closeResources()
}

func (reader *Reader) closeSubscription(subscription *fileSubscription) {
	reader.mu.Lock()
	subscriptions := reader.subscriptions[subscription.key]
	if _, ok := subscriptions[subscription]; ok {
		delete(subscriptions, subscription)
		if len(subscriptions) == 0 {
			delete(reader.subscriptions, subscription.key)
		}
		subscription.terminate(fileError(configcenter.CodeSubscriptionClosed, subscription.key, configcenter.OperationNext))
	}
	reader.mu.Unlock()
}

func snapshotForState(key configcenter.Key, state fileState, revision configcenter.Revision) (configcenter.Snapshot, error) {
	if !state.present {
		return configcenter.NewMissingSnapshot(key, revision)
	}
	return configcenter.NewPresentSnapshot(key, state.content, revision)
}

var _ configcenter.DynamicConfiguration = (*Reader)(nil)

func closeRootIgnoringError(root *os.Root) {
	if root != nil {
		_ = root.Close()
	}
}

func closeWatcherIgnoringError(watcher sourceWatcher) {
	if watcher != nil {
		_ = watcher.Close()
	}
}
