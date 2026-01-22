// Package watcher provides file watching capabilities for the transcription service.
package watcher

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unsafe"

	"golang.org/x/sys/unix"
)

// FileEvent represents a detected file.
type FileEvent struct {
	Path      string
	Size      int64
	Timestamp time.Time
}

// FileWatcher detects new files in a directory.
type FileWatcher interface {
	Watch(ctx context.Context, dir string, patterns []string) (<-chan FileEvent, error)
	Stop() error
}

// InotifyWatcher implements FileWatcher using Linux inotify.
type InotifyWatcher struct {
	fd       int
	wd       int
	patterns []string
	stopCh   chan struct{}
	stopped  bool
}

// NewInotifyWatcher creates a new inotify-based file watcher.
func NewInotifyWatcher() (*InotifyWatcher, error) {
	fd, err := unix.InotifyInit1(unix.IN_CLOEXEC | unix.IN_NONBLOCK)
	if err != nil {
		return nil, err
	}

	return &InotifyWatcher{
		fd:     fd,
		stopCh: make(chan struct{}),
	}, nil
}

// Watch starts watching the specified directory for files matching the patterns.
func (w *InotifyWatcher) Watch(ctx context.Context, dir string, patterns []string) (<-chan FileEvent, error) {
	// Add watch for the directory
	wd, err := unix.InotifyAddWatch(w.fd, dir, unix.IN_CLOSE_WRITE|unix.IN_MOVED_TO)
	if err != nil {
		return nil, err
	}
	w.wd = wd
	w.patterns = patterns

	events := make(chan FileEvent, 100)

	go w.readEvents(ctx, dir, events)

	return events, nil
}

// Stop stops the watcher and releases resources.
func (w *InotifyWatcher) Stop() error {
	if w.stopped {
		return nil
	}
	w.stopped = true
	close(w.stopCh)

	if w.wd != 0 {
		unix.InotifyRmWatch(w.fd, uint32(w.wd))
	}
	return unix.Close(w.fd)
}

func (w *InotifyWatcher) readEvents(ctx context.Context, dir string, events chan<- FileEvent) {
	defer close(events)

	buf := make([]byte, 4096)

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.stopCh:
			return
		default:
		}

		n, err := unix.Read(w.fd, buf)
		if err != nil {
			if err == unix.EAGAIN || err == unix.EWOULDBLOCK {
				// No events available, sleep briefly and retry
				time.Sleep(10 * time.Millisecond)
				continue
			}
			return
		}

		if n < unix.SizeofInotifyEvent {
			continue
		}

		// Parse inotify events
		offset := 0
		for offset < n {
			event := (*unix.InotifyEvent)(unsafe.Pointer(&buf[offset]))
			nameLen := int(event.Len)

			if nameLen > 0 {
				nameBytes := buf[offset+unix.SizeofInotifyEvent : offset+unix.SizeofInotifyEvent+nameLen]
				name := strings.TrimRight(string(nameBytes), "\x00")

				if w.matchesPatterns(name) {
					fullPath := filepath.Join(dir, name)
					info, err := os.Stat(fullPath)
					if err == nil {
						events <- FileEvent{
							Path:      fullPath,
							Size:      info.Size(),
							Timestamp: time.Now(),
						}
					}
				}
			}

			offset += unix.SizeofInotifyEvent + nameLen
		}
	}
}

func (w *InotifyWatcher) matchesPatterns(name string) bool {
	if len(w.patterns) == 0 {
		return true
	}

	for _, pattern := range w.patterns {
		matched, err := filepath.Match(pattern, name)
		if err == nil && matched {
			return true
		}
	}
	return false
}
