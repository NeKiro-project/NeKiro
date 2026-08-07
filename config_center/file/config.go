// Package file implements the deterministic local File Config Center
// provider. It is intentionally separate from the provider-neutral core.
package file

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"

	configcenter "github.com/NeKiro-project/NeKiro/config_center"
)

// ReaderConfig contains every required input for a File reader. No field is
// inferred when omitted or zero.
type ReaderConfig struct {
	Root               string
	MaxPayloadBytes    int64
	SubscriptionBuffer int
}

// PublisherConfig contains every required input for a File publisher. Reader
// and publisher roots are opened independently even when they name the same
// directory.
type PublisherConfig struct {
	Root            string
	MaxPayloadBytes int64
	FileMode        fs.FileMode
}

const fileProvider = configcenter.ProviderID("file")

// fileOperations is intentionally package-private. Production uses the fixed
// platform implementation; package-local tests can make pathname races
// deterministic without adding a production control surface.
type fileOperations struct {
	lstat    func(string) (fs.FileInfo, error)
	openRoot func(string) (*os.Root, error)
	rootStat func(*os.Root, string) (fs.FileInfo, error)

	afterInitialRootLstat  func()
	afterWatcherAdd        func()
	beforePublisherCommit  func()
	beforePublisherSuccess func()
}

func productionFileOperations() fileOperations {
	return fileOperations{
		lstat:    os.Lstat,
		openRoot: os.OpenRoot,
		rootStat: func(root *os.Root, name string) (fs.FileInfo, error) {
			return root.Stat(name)
		},
	}
}

func validatePlatform(operation configcenter.Operation) error {
	if runtime.GOOS == "js" || runtime.GOOS == "plan9" {
		return configcenter.NewError(configcenter.CodeUnsupported, configcenter.ErrorDetails{
			Provider:  fileProvider,
			Operation: operation,
		})
	}
	return nil
}

func validateReaderConfig(config ReaderConfig) error {
	if err := validatePlatform(configcenter.OperationObserve); err != nil {
		return err
	}
	if !validRootPath(config.Root) || config.MaxPayloadBytes <= 0 || config.SubscriptionBuffer <= 0 {
		return configcenter.NewError(configcenter.CodeInvalid, configcenter.ErrorDetails{
			Provider:  fileProvider,
			Operation: configcenter.OperationObserve,
		})
	}
	return nil
}

func validatePublisherConfig(config PublisherConfig) error {
	if err := validatePlatform(configcenter.OperationPublish); err != nil {
		return err
	}
	if !validRootPath(config.Root) || config.MaxPayloadBytes <= 0 || !validFileMode(config.FileMode) {
		return configcenter.NewError(configcenter.CodeInvalid, configcenter.ErrorDetails{
			Provider:  fileProvider,
			Operation: configcenter.OperationPublish,
		})
	}
	return nil
}

func validRootPath(path string) bool {
	return path != "" && filepath.IsAbs(path) && filepath.Clean(path) == path
}

func validFileMode(mode fs.FileMode) bool {
	if mode == 0 || mode.Perm() != mode {
		return false
	}
	if runtime.GOOS == "windows" {
		// Windows exposes only the read-only attribute through os.Chmod. These
		// are the only exact permission projections the File provider promises.
		return mode == 0o666 || mode == 0o444
	}
	return true
}

func openPinnedRoot(path string, operation configcenter.Operation) (*os.Root, fs.FileInfo, error) {
	return openPinnedRootWithOperations(path, operation, productionFileOperations())
}

func openPinnedRootWithOperations(path string, operation configcenter.Operation, operations fileOperations) (*os.Root, fs.FileInfo, error) {
	before, err := operations.lstat(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) || os.IsNotExist(err) {
			return nil, nil, configcenter.NewError(configcenter.CodeUnavailable, configcenter.ErrorDetails{
				Provider:  fileProvider,
				Operation: operation,
			})
		}
		return nil, nil, mapFilesystemError(operation, err)
	}
	if !isNonSymlinkDirectory(before) {
		return nil, nil, configcenter.NewError(configcenter.CodeInvalid, configcenter.ErrorDetails{
			Provider:  fileProvider,
			Operation: operation,
		})
	}
	// Windows FileInfo loads its file ID lazily from the saved pathname when
	// os.SameFile is first called. Materialize it before any pathname
	// substitution window so the identity cannot follow a replacement.
	_ = os.SameFile(before, before)
	invokeFileHook(operations.afterInitialRootLstat)
	root, err := operations.openRoot(path)
	if err != nil {
		return nil, nil, mapRootOpenFailure(path, before, operation, operations, err)
	}
	opened, err := operations.rootStat(root, ".")
	if err != nil {
		closeRootIgnoringError(root)
		return nil, nil, mapFilesystemError(operation, err)
	}
	after, err := operations.lstat(path)
	if err != nil {
		closeRootIgnoringError(root)
		return nil, nil, mapFilesystemError(operation, err)
	}
	if !samePinnedRoot(before, opened, after) {
		closeRootIgnoringError(root)
		return nil, nil, configcenter.NewError(configcenter.CodeUnsafeState, configcenter.ErrorDetails{
			Provider:  fileProvider,
			Operation: operation,
		})
	}
	return root, opened, nil
}

// mapRootOpenFailure classifies a pathname substitution observed after the
// initial acceptance separately from an ordinary OpenRoot I/O failure. The
// extra Lstat observes the current path once; it does not retry or reopen.
func mapRootOpenFailure(path string, before fs.FileInfo, operation configcenter.Operation, operations fileOperations, openErr error) error {
	current, err := operations.lstat(path)
	if err != nil {
		return mapFilesystemError(operation, err)
	}
	if !isNonSymlinkDirectory(current) || !os.SameFile(before, current) {
		return configcenter.NewError(configcenter.CodeUnsafeState, configcenter.ErrorDetails{
			Provider:  fileProvider,
			Operation: operation,
		})
	}
	return mapFilesystemError(operation, openErr)
}

func rootIdentityCode(path string, root *os.Root, pinned fs.FileInfo, operations fileOperations) configcenter.Code {
	opened, err := operations.rootStat(root, ".")
	if err != nil {
		return filesystemErrorCode(err)
	}
	current, err := operations.lstat(path)
	if err != nil {
		return filesystemErrorCode(err)
	}
	if !samePinnedRoot(pinned, opened, current) {
		return configcenter.CodeUnsafeState
	}
	return ""
}

func samePinnedRoot(before, opened, after fs.FileInfo) bool {
	return isNonSymlinkDirectory(before) &&
		isNonSymlinkDirectory(opened) &&
		isNonSymlinkDirectory(after) &&
		os.SameFile(before, opened) &&
		os.SameFile(before, after)
}

func isNonSymlinkDirectory(info fs.FileInfo) bool {
	return info != nil && info.Mode()&os.ModeSymlink == 0 && info.IsDir()
}

func invokeFileHook(hook func()) {
	if hook != nil {
		hook()
	}
}

func mapFilesystemError(operation configcenter.Operation, err error) error {
	code := filesystemErrorCode(err)
	return configcenter.NewError(code, configcenter.ErrorDetails{
		Provider:  fileProvider,
		Operation: operation,
	})
}

func filesystemErrorCode(err error) configcenter.Code {
	if errors.Is(err, fs.ErrPermission) || os.IsPermission(err) {
		return configcenter.CodeUnauthorized
	}
	return configcenter.CodeUnavailable
}

func fileError(code configcenter.Code, key configcenter.Key, operation configcenter.Operation) error {
	return configcenter.NewError(code, configcenter.ErrorDetails{
		Provider:  fileProvider,
		Key:       key,
		Operation: operation,
	})
}

func watchInterruptedError(key configcenter.Key, operation configcenter.Operation, cause configcenter.Code) error {
	details := configcenter.ErrorDetails{
		Provider:  fileProvider,
		Key:       key,
		Operation: operation,
		CauseKind: cause,
	}
	return configcenter.NewError(configcenter.CodeWatchInterrupted, details)
}
