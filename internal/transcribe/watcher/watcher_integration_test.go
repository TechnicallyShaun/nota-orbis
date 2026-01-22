//go:build linux

package watcher

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestInotifyWatcher_DetectsNewFile(t *testing.T) {
	tmpDir := t.TempDir()

	watcher, err := NewInotifyWatcher()
	if err != nil {
		t.Fatalf("failed to create watcher: %v", err)
	}
	defer watcher.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	events, err := watcher.Watch(ctx, tmpDir, []string{"*.txt"})
	if err != nil {
		t.Fatalf("failed to start watch: %v", err)
	}

	// Give the watcher time to set up
	time.Sleep(50 * time.Millisecond)

	// Create a new file
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("hello"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Wait for the event
	select {
	case event := <-events:
		if event.Path != testFile {
			t.Errorf("expected path %s, got %s", testFile, event.Path)
		}
		if event.Size != 5 {
			t.Errorf("expected size 5, got %d", event.Size)
		}
	case <-ctx.Done():
		t.Fatal("timeout waiting for file event")
	}
}

func TestInotifyWatcher_IgnoresNonMatchingPatterns(t *testing.T) {
	tmpDir := t.TempDir()

	watcher, err := NewInotifyWatcher()
	if err != nil {
		t.Fatalf("failed to create watcher: %v", err)
	}
	defer watcher.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	events, err := watcher.Watch(ctx, tmpDir, []string{"*.m4a"})
	if err != nil {
		t.Fatalf("failed to start watch: %v", err)
	}

	// Give the watcher time to set up
	time.Sleep(50 * time.Millisecond)

	// Create a file that doesn't match the pattern
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("hello"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Should not receive any event
	select {
	case event := <-events:
		t.Errorf("unexpected event for non-matching file: %v", event)
	case <-time.After(500 * time.Millisecond):
		// Expected: no event for non-matching pattern
	}
}

func TestInotifyWatcher_DetectsMovedFile(t *testing.T) {
	tmpDir := t.TempDir()
	srcDir := t.TempDir()

	watcher, err := NewInotifyWatcher()
	if err != nil {
		t.Fatalf("failed to create watcher: %v", err)
	}
	defer watcher.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	events, err := watcher.Watch(ctx, tmpDir, []string{"*.m4a"})
	if err != nil {
		t.Fatalf("failed to start watch: %v", err)
	}

	// Give the watcher time to set up
	time.Sleep(50 * time.Millisecond)

	// Create a file in source directory and move it to watched directory
	srcFile := filepath.Join(srcDir, "audio.m4a")
	if err := os.WriteFile(srcFile, []byte("fake audio content"), 0644); err != nil {
		t.Fatalf("failed to create source file: %v", err)
	}

	dstFile := filepath.Join(tmpDir, "audio.m4a")
	if err := os.Rename(srcFile, dstFile); err != nil {
		t.Fatalf("failed to move file: %v", err)
	}

	// Wait for the event
	select {
	case event := <-events:
		if event.Path != dstFile {
			t.Errorf("expected path %s, got %s", dstFile, event.Path)
		}
	case <-ctx.Done():
		t.Fatal("timeout waiting for moved file event")
	}
}

func TestInotifyWatcher_StopCleansUp(t *testing.T) {
	tmpDir := t.TempDir()

	watcher, err := NewInotifyWatcher()
	if err != nil {
		t.Fatalf("failed to create watcher: %v", err)
	}

	ctx := context.Background()
	events, err := watcher.Watch(ctx, tmpDir, nil)
	if err != nil {
		t.Fatalf("failed to start watch: %v", err)
	}

	// Stop the watcher
	if err := watcher.Stop(); err != nil {
		t.Errorf("stop failed: %v", err)
	}

	// Events channel should be closed
	select {
	case _, ok := <-events:
		if ok {
			t.Error("expected events channel to be closed")
		}
	case <-time.After(time.Second):
		t.Error("events channel not closed after stop")
	}

	// Double stop should not error
	if err := watcher.Stop(); err != nil {
		t.Errorf("double stop failed: %v", err)
	}
}
