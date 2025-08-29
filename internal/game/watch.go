package game

import (
	"os"
	"time"
)

// FileWatcher polls file modification times and triggers a callback on change.
// It uses only the standard library for simplicity.
type FileWatcher struct {
	Paths     []string
	Interval  time.Duration
	onChange  func(string) // called with path that changed
	stopCh    chan struct{}
	lastMTime map[string]time.Time
}

// NewFileWatcher creates a watcher for given paths and interval.
func NewFileWatcher(paths []string, interval time.Duration, onChange func(string)) *FileWatcher {
	return &FileWatcher{
		Paths:     paths,
		Interval:  interval,
		onChange:  onChange,
		stopCh:    make(chan struct{}),
		lastMTime: make(map[string]time.Time),
	}
}

// Start begins polling in a goroutine.
func (w *FileWatcher) Start() {
	ticker := time.NewTicker(w.Interval)
	go func() {
		defer ticker.Stop()
		// prime cache
		w.scanAll(true)
		for {
			select {
			case <-ticker.C:
				w.scanAll(false)
			case <-w.stopCh:
				return
			}
		}
	}()
}

// Stop terminates the watcher.
func (w *FileWatcher) Stop() {
	close(w.stopCh)
}

// scanAll checks mtimes and invokes onChange for files that changed since last scan.
func (w *FileWatcher) scanAll(prime bool) {
	for _, p := range w.Paths {
		fi, err := os.Stat(p)
		if err != nil {
			// if file missing, treat mtime as zero and keep going
			continue
		}
		mt := fi.ModTime()
		last, ok := w.lastMTime[p]
		if !ok {
			// first time seeing this file
			w.lastMTime[p] = mt
			continue
		}
		if mt.After(last) {
			w.lastMTime[p] = mt
			if !prime && w.onChange != nil {
				w.onChange(p)
			}
		}
	}
}
