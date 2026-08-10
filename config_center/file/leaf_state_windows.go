//go:build windows

package file

import (
	"path/filepath"
	"syscall"
)

// leafStateAfterAccessFailure preserves provider-neutral missing and unsafe
// outcomes when os.Root reports a Windows permission or sharing error during
// pathname replacement. GetFileAttributes inspects the pathname object; it
// does not open or follow the target and is not a read retry or fallback.
func leafStateAfterAccessFailure(rootPath, leaf string) leafFailureState {
	path, err := syscall.UTF16PtrFromString(filepath.Join(rootPath, leaf))
	if err != nil {
		return leafFailureUnknown
	}
	attributes, err := syscall.GetFileAttributes(path)
	if err != nil {
		if err == syscall.ERROR_FILE_NOT_FOUND || err == syscall.ERROR_PATH_NOT_FOUND {
			return leafFailureMissing
		}
		return leafFailureUnknown
	}
	if attributes&(syscall.FILE_ATTRIBUTE_REPARSE_POINT|syscall.FILE_ATTRIBUTE_DIRECTORY) != 0 {
		return leafFailureUnsafe
	}
	return leafFailureUnknown
}
