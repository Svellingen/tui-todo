package model

import (
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

// settleDelay is how long to wait for a burst of filesystem events to stop
// before treating a save as finished. Editors emit several events for one
// save, and a direct append can be observed mid-write.
const settleDelay = 40 * time.Millisecond

// fileWatcher reports changes to a single file.
type fileWatcher struct {
	fsw  *fsnotify.Watcher
	path string
}

// newFileWatcher starts watching the directory holding path.
//
// It watches the directory rather than the file itself on purpose: editors
// that save atomically write a temp file and rename it over the original, so a
// watch on the file would be left pointing at a discarded inode after the
// first save, and every later save would go unnoticed.
func newFileWatcher(path string) (*fileWatcher, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	if err := fsw.Add(filepath.Dir(abs)); err != nil {
		fsw.Close()
		return nil, err
	}
	return &fileWatcher{fsw: fsw, path: abs}, nil
}

// wait blocks until the watched file changes, then swallows the rest of the
// burst so that one save produces one reload. It reports false once the
// watcher is closed.
func (w *fileWatcher) wait() bool {
	for {
		select {
		case ev, ok := <-w.fsw.Events:
			if !ok {
				return false
			}
			if !w.relevant(ev) {
				continue
			}
			w.settle()
			return true
		case _, ok := <-w.fsw.Errors:
			if !ok {
				return false
			}
			// A dropped event is not fatal: the next one still arrives, and
			// the reload compares against the file rather than the event.
		}
	}
}

// settle drains events until the directory has been quiet for settleDelay.
func (w *fileWatcher) settle() {
	timer := time.NewTimer(settleDelay)
	defer timer.Stop()
	for {
		select {
		case _, ok := <-w.fsw.Events:
			if !ok {
				return
			}
			timer.Reset(settleDelay)
		case <-timer.C:
			return
		}
	}
}

// relevant reports whether an event concerns the watched file's contents.
// The directory watch also surfaces its siblings, which are ignored.
func (w *fileWatcher) relevant(ev fsnotify.Event) bool {
	if filepath.Clean(ev.Name) != w.path {
		return false
	}
	return ev.Op&(fsnotify.Create|fsnotify.Write|fsnotify.Remove|fsnotify.Rename) != 0
}

func (w *fileWatcher) close() {
	if w != nil && w.fsw != nil {
		w.fsw.Close()
	}
}
