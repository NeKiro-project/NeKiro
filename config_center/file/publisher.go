package file

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"io/fs"
	"os"
	"sync"

	configcenter "github.com/NeKiro-project/NeKiro/config_center"
)

// Publisher is the separately injected File administrative capability. It is
// not embedded in Reader and never exposes read/observe methods.
type Publisher struct {
	root            *os.Root
	rootPath        string
	rootIdentity    fs.FileInfo
	operations      fileOperations
	maxPayloadBytes int64
	fileMode        fs.FileMode

	mu        sync.Mutex
	closed    bool
	closeDone chan struct{}
	closeErr  error
}

// OpenPublisher independently pins the explicit configured root.
func OpenPublisher(config PublisherConfig) (*Publisher, error) {
	return openPublisherWithOperations(config, productionFileOperations())
}

func openPublisherWithOperations(config PublisherConfig, operations fileOperations) (*Publisher, error) {
	if err := validatePublisherConfig(config); err != nil {
		return nil, err
	}
	root, rootIdentity, err := openPinnedRootWithOperations(config.Root, configcenter.OperationPublish, operations)
	if err != nil {
		return nil, err
	}
	return &Publisher{
		root:            root,
		rootPath:        config.Root,
		rootIdentity:    rootIdentity,
		operations:      operations,
		maxPayloadBytes: config.MaxPayloadBytes,
		fileMode:        config.FileMode,
		closeDone:       make(chan struct{}),
	}, nil
}

// Publish replaces one mapped state atomically. Input is copied before any
// write, then written to a same-root temporary leaf, synced, closed, and only
// then renamed onto the already safety-checked target.
func (publisher *Publisher) Publish(ctx context.Context, key configcenter.Key, value []byte) (result error) {
	if err := contextError(ctx, key, configcenter.OperationPublish); err != nil {
		return err
	}
	if !key.Valid() {
		return fileError(configcenter.CodeInvalid, key, configcenter.OperationPublish)
	}
	payload := bytes.Clone(value)
	if int64(len(payload)) > publisher.maxPayloadBytes {
		return fileError(configcenter.CodePayloadTooLarge, key, configcenter.OperationPublish)
	}

	publisher.mu.Lock()
	defer publisher.mu.Unlock()
	if publisher.closed {
		return fileError(configcenter.CodePublisherClosed, key, configcenter.OperationPublish)
	}
	if err := contextError(ctx, key, configcenter.OperationPublish); err != nil {
		return err
	}
	if err := publisher.validateRootIdentityLocked(key, configcenter.OperationPublish); err != nil {
		return err
	}
	leaf, _, err := publisher.inspectTargetLocked(key, configcenter.OperationPublish)
	if err != nil {
		return err
	}
	temporaryLeaf, err := publisher.temporaryLeaf()
	if err != nil {
		return fileError(configcenter.CodeUnavailable, key, configcenter.OperationPublish)
	}
	temporary, err := publisher.root.OpenFile(temporaryLeaf, os.O_WRONLY|os.O_CREATE|os.O_EXCL, publisher.fileMode)
	if err != nil {
		return mapLeafFilesystemError(key, configcenter.OperationPublish, err)
	}
	published := false
	defer func() {
		if temporary != nil {
			if closeErr := temporary.Close(); closeErr != nil {
				if result == nil {
					result = fileError(configcenter.CodeUnavailable, key, configcenter.OperationPublish)
				}
			}
		}
		if !published {
			if cleanupErr := publisher.root.Remove(temporaryLeaf); cleanupErr != nil && !errors.Is(cleanupErr, fs.ErrNotExist) && !os.IsNotExist(cleanupErr) {
				if result == nil {
					result = fileError(configcenter.CodeUnavailable, key, configcenter.OperationPublish)
				}
			}
		}
	}()

	if err := temporary.Chmod(publisher.fileMode); err != nil {
		return mapLeafFilesystemError(key, configcenter.OperationPublish, err)
	}
	if err := contextError(ctx, key, configcenter.OperationPublish); err != nil {
		return err
	}
	count, writeErr := temporary.Write(payload)
	if writeErr != nil {
		return mapLeafFilesystemError(key, configcenter.OperationPublish, writeErr)
	}
	if count != len(payload) {
		return mapLeafFilesystemError(key, configcenter.OperationPublish, io.ErrShortWrite)
	}
	if err := temporary.Sync(); err != nil {
		return mapLeafFilesystemError(key, configcenter.OperationPublish, err)
	}
	if err := temporary.Close(); err != nil {
		temporary = nil
		return mapLeafFilesystemError(key, configcenter.OperationPublish, err)
	}
	temporary = nil
	if err := contextError(ctx, key, configcenter.OperationPublish); err != nil {
		return err
	}
	invokeFileHook(publisher.operations.beforePublisherCommit)
	if err := publisher.validateRootIdentityLocked(key, configcenter.OperationPublish); err != nil {
		return err
	}
	if err := publisher.root.Rename(temporaryLeaf, leaf); err != nil {
		return mapLeafFilesystemError(key, configcenter.OperationPublish, err)
	}
	published = true
	invokeFileHook(publisher.operations.beforePublisherSuccess)
	if err := publisher.validateRootIdentityLocked(key, configcenter.OperationPublish); err != nil {
		return err
	}
	return nil
}

// Delete explicitly removes one mapped state. An absent leaf is not treated as
// success and does not restore any default value.
func (publisher *Publisher) Delete(ctx context.Context, key configcenter.Key) error {
	if err := contextError(ctx, key, configcenter.OperationDelete); err != nil {
		return err
	}
	if !key.Valid() {
		return fileError(configcenter.CodeInvalid, key, configcenter.OperationDelete)
	}
	publisher.mu.Lock()
	defer publisher.mu.Unlock()
	if publisher.closed {
		return fileError(configcenter.CodePublisherClosed, key, configcenter.OperationDelete)
	}
	if err := contextError(ctx, key, configcenter.OperationDelete); err != nil {
		return err
	}
	if err := publisher.validateRootIdentityLocked(key, configcenter.OperationDelete); err != nil {
		return err
	}
	leaf, exists, err := publisher.inspectTargetLocked(key, configcenter.OperationDelete)
	if err != nil {
		return err
	}
	if !exists {
		return fileError(configcenter.CodeMissing, key, configcenter.OperationDelete)
	}
	invokeFileHook(publisher.operations.beforePublisherCommit)
	if err := publisher.validateRootIdentityLocked(key, configcenter.OperationDelete); err != nil {
		return err
	}
	if err := publisher.root.Remove(leaf); err != nil {
		if errors.Is(err, fs.ErrNotExist) || os.IsNotExist(err) {
			return fileError(configcenter.CodeMissing, key, configcenter.OperationDelete)
		}
		return mapLeafFilesystemError(key, configcenter.OperationDelete, err)
	}
	invokeFileHook(publisher.operations.beforePublisherSuccess)
	if err := publisher.validateRootIdentityLocked(key, configcenter.OperationDelete); err != nil {
		return err
	}
	return nil
}

func (publisher *Publisher) validateRootIdentityLocked(key configcenter.Key, operation configcenter.Operation) error {
	if code := rootIdentityCode(publisher.rootPath, publisher.root, publisher.rootIdentity, publisher.operations); code != "" {
		return fileError(code, key, operation)
	}
	return nil
}

// Close releases the pinned publisher root. It is idempotent and never reopens
// a moved, deleted, or replacement directory.
func (publisher *Publisher) Close() error {
	publisher.mu.Lock()
	if publisher.closed {
		done := publisher.closeDone
		publisher.mu.Unlock()
		<-done
		publisher.mu.Lock()
		err := publisher.closeErr
		publisher.mu.Unlock()
		return err
	}
	publisher.closed = true
	root := publisher.root
	publisher.mu.Unlock()

	rootErr := root.Close()
	var closeErr error
	if rootErr != nil {
		closeErr = fileError(configcenter.CodeUnavailable, configcenter.Key{}, configcenter.OperationClose)
	}
	publisher.mu.Lock()
	publisher.closeErr = closeErr
	close(publisher.closeDone)
	publisher.mu.Unlock()
	return closeErr
}

func (publisher *Publisher) inspectTargetLocked(key configcenter.Key, operation configcenter.Operation) (string, bool, error) {
	leaf, err := MapKey(key)
	if err != nil {
		return "", false, fileError(configcenter.CodeInvalid, key, operation)
	}
	info, err := publisher.root.Lstat(leaf)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) || os.IsNotExist(err) {
			return leaf, false, nil
		}
		return "", false, mapLeafFilesystemError(key, operation, err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return "", false, fileError(configcenter.CodeUnsafeState, key, operation)
	}
	return leaf, true, nil
}

func (publisher *Publisher) temporaryLeaf() (string, error) {
	var entropy [16]byte
	if _, err := rand.Read(entropy[:]); err != nil {
		return "", err
	}
	// A dot-prefixed temporary name cannot satisfy the cfg-v1-...value mapping.
	return ".config-center-tmp-v1-" + base64.RawURLEncoding.EncodeToString(entropy[:]), nil
}

var _ configcenter.ConfigurationPublisher = (*Publisher)(nil)
