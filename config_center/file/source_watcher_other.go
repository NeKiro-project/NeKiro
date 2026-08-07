//go:build !darwin

package file

import (
	"io/fs"

	"github.com/fsnotify/fsnotify"
)

type fsnotifySourceWatcher struct{ watcher *fsnotify.Watcher }

func openSourceWatcher(root string) (sourceWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	if err := watcher.Add(root); err != nil {
		_ = watcher.Close()
		return nil, err
	}
	return fsnotifySourceWatcher{watcher: watcher}, nil
}

func (watcher fsnotifySourceWatcher) Events() <-chan fsnotify.Event { return watcher.watcher.Events }
func (watcher fsnotifySourceWatcher) Errors() <-chan error          { return watcher.watcher.Errors }
func (watcher fsnotifySourceWatcher) Close() error                  { return watcher.watcher.Close() }
func (watcher fsnotifySourceWatcher) WatchList() []string           { return watcher.watcher.WatchList() }
func (watcher fsnotifySourceWatcher) Identity() fs.FileInfo         { return nil }
func (watcher fsnotifySourceWatcher) AllowsNotExistReconciliation() bool {
	return true
}
