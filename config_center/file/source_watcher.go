package file

import (
	"io/fs"
	"os"

	"github.com/fsnotify/fsnotify"
)

type sourceWatcher interface {
	Events() <-chan fsnotify.Event
	Errors() <-chan error
	Close() error
	WatchList() []string
	Identity() fs.FileInfo
	AllowsNotExistReconciliation() bool
}

func watcherIdentityMatches(watcher sourceWatcher, pinned fs.FileInfo) bool {
	identity := watcher.Identity()
	return identity == nil || (isNonSymlinkDirectory(identity) && os.SameFile(identity, pinned))
}
