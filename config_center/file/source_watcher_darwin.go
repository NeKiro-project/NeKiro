//go:build darwin

package file

import (
	"errors"
	"io/fs"
	"os"
	"sync"
	"syscall"

	"github.com/fsnotify/fsnotify"
	"golang.org/x/sys/unix"
)

const kqueueWakeIdent = 1

type keventFunc func(int, []unix.Kevent_t, []unix.Kevent_t, *unix.Timespec) (int, error)

type kqueueRootWatcher struct {
	rootPath string
	rootFile *os.File
	identity fs.FileInfo
	kqueue   int
	events   chan fsnotify.Event
	errors   chan error
	closing  chan struct{}
	done     chan struct{}
	close    sync.Once
}

func openSourceWatcher(rootPath string) (sourceWatcher, error) {
	rootFD, err := unix.Open(rootPath, unix.O_EVTONLY|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	rootFile := os.NewFile(uintptr(rootFD), rootPath)
	identity, err := rootFile.Stat()
	if err != nil {
		_ = rootFile.Close()
		return nil, err
	}
	kqueue, err := unix.Kqueue()
	if err != nil {
		_ = rootFile.Close()
		return nil, err
	}
	watcher := &kqueueRootWatcher{
		rootPath: rootPath,
		rootFile: rootFile,
		identity: identity,
		kqueue:   kqueue,
		events:   make(chan fsnotify.Event),
		errors:   make(chan error),
		closing:  make(chan struct{}),
		done:     make(chan struct{}),
	}
	changes := []unix.Kevent_t{{}, {}}
	unix.SetKevent(&changes[0], rootFD, unix.EVFILT_VNODE, unix.EV_ADD|unix.EV_CLEAR)
	changes[0].Fflags = unix.NOTE_WRITE | unix.NOTE_DELETE | unix.NOTE_RENAME | unix.NOTE_ATTRIB | unix.NOTE_REVOKE
	unix.SetKevent(&changes[1], kqueueWakeIdent, unix.EVFILT_USER, unix.EV_ADD|unix.EV_CLEAR)
	if _, err := completeKevent(unix.Kevent, kqueue, changes, nil, nil); err != nil {
		_ = unix.Close(kqueue)
		_ = rootFile.Close()
		return nil, err
	}
	go watcher.run(rootFD)
	return watcher, nil
}

func (watcher *kqueueRootWatcher) Events() <-chan fsnotify.Event { return watcher.events }
func (watcher *kqueueRootWatcher) Errors() <-chan error          { return watcher.errors }
func (watcher *kqueueRootWatcher) WatchList() []string           { return []string{watcher.rootPath} }
func (watcher *kqueueRootWatcher) Identity() fs.FileInfo         { return watcher.identity }
func (watcher *kqueueRootWatcher) AllowsNotExistReconciliation() bool {
	return false
}

func (watcher *kqueueRootWatcher) Close() (closeErr error) {
	watcher.close.Do(func() {
		close(watcher.closing)
		change := unix.Kevent_t{}
		unix.SetKevent(&change, kqueueWakeIdent, unix.EVFILT_USER, 0)
		change.Fflags = unix.NOTE_TRIGGER
		if _, err := completeKevent(unix.Kevent, watcher.kqueue, []unix.Kevent_t{change}, nil, nil); err != nil {
			closeErr = err
		}
		<-watcher.done
		if err := unix.Close(watcher.kqueue); err != nil && closeErr == nil {
			closeErr = err
		}
		if err := watcher.rootFile.Close(); err != nil && closeErr == nil {
			closeErr = err
		}
	})
	return closeErr
}

func (watcher *kqueueRootWatcher) run(rootFD int) {
	defer close(watcher.done)
	defer close(watcher.events)
	defer close(watcher.errors)
	for {
		changes := make([]unix.Kevent_t, 1)
		count, err := completeKevent(unix.Kevent, watcher.kqueue, nil, changes, nil)
		if err != nil {
			watcher.sendError(err)
			return
		}
		for _, change := range changes[:count] {
			if change.Filter == unix.EVFILT_USER && change.Ident == kqueueWakeIdent {
				return
			}
			if int(change.Ident) != rootFD {
				watcher.sendError(errors.New("unexpected kqueue source"))
				return
			}
			if change.Flags&unix.EV_ERROR != 0 {
				watcher.sendError(syscall.Errno(change.Data))
				return
			}
			if change.Flags&unix.EV_EOF != 0 {
				watcher.sendError(errors.New("kqueue eof"))
				return
			}
			op := fsnotify.Op(0)
			if change.Fflags&unix.NOTE_WRITE != 0 {
				op |= fsnotify.Write
			}
			if change.Fflags&unix.NOTE_DELETE != 0 {
				op |= fsnotify.Remove
			}
			if change.Fflags&unix.NOTE_RENAME != 0 {
				op |= fsnotify.Rename
			}
			if change.Fflags&unix.NOTE_ATTRIB != 0 {
				op |= fsnotify.Chmod
			}
			if change.Fflags&unix.NOTE_REVOKE != 0 {
				op |= fsnotify.Remove
			}
			if op == 0 {
				continue
			}
			select {
			case watcher.events <- fsnotify.Event{Name: watcher.rootPath, Op: op}:
			case <-watcher.closing:
				return
			}
		}
	}
}

// completeKevent repeats only an interrupted kernel call against the already
// registered queue. EINTR reports no provider event or source fault, so this is
// not provider recovery, resubscription, or an alternate watcher.
func completeKevent(call keventFunc, kqueue int, changes, events []unix.Kevent_t, timeout *unix.Timespec) (int, error) {
	for {
		count, err := call(kqueue, changes, events, timeout)
		if !errors.Is(err, unix.EINTR) {
			return count, err
		}
	}
}

func (watcher *kqueueRootWatcher) sendError(err error) {
	select {
	case watcher.errors <- err:
	case <-watcher.closing:
	}
}
