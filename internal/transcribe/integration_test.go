//go:build linux

// Package transcribe contains integration tests for the full file watcher pipeline.
package transcribe

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/TechnicallyShaun/nota-orbis/internal/transcribe/metadata"
	"github.com/TechnicallyShaun/nota-orbis/internal/transcribe/stabilizer"
	"github.com/TechnicallyShaun/nota-orbis/internal/transcribe/watcher"
)

// TestFullFlow_FileDropStableMetadata tests the complete pipeline:
// file drop -> detect -> stabilize -> extract metadata
func TestFullFlow_FileDropStableMetadata(t *testing.T) {
	watchDir := t.TempDir()
	srcDir := t.TempDir()

	// Create watcher
	w, err := watcher.NewInotifyWatcher()
	if err != nil {
		t.Fatalf("failed to create watcher: %v", err)
	}
	defer w.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	events, err := w.Watch(ctx, watchDir, []string{"*.m4a"})
	if err != nil {
		t.Fatalf("failed to start watch: %v", err)
	}

	// Create stabilizer
	stab := stabilizer.NewPollStabilizer(50*time.Millisecond, 3)

	// Give the watcher time to set up
	time.Sleep(100 * time.Millisecond)

	// Simulate file drop: create M4A in source then move to watch dir
	srcFile := filepath.Join(srcDir, "voice-note.m4a")
	creationTime := time.Date(2026, 1, 22, 14, 30, 0, 0, time.UTC)
	durationSeconds := uint32(90)

	if err := createTestM4A(srcFile, creationTime, durationSeconds); err != nil {
		t.Fatalf("failed to create test M4A: %v", err)
	}

	// Move file to watch directory (simulates Syncthing completing a sync)
	dstFile := filepath.Join(watchDir, "voice-note.m4a")
	if err := os.Rename(srcFile, dstFile); err != nil {
		t.Fatalf("failed to move file: %v", err)
	}

	// Wait for file detection
	var detectedPath string
	select {
	case event := <-events:
		detectedPath = event.Path
		t.Logf("detected file: %s (size: %d)", event.Path, event.Size)
	case <-ctx.Done():
		t.Fatal("timeout waiting for file detection")
	}

	if detectedPath != dstFile {
		t.Errorf("detected wrong path: expected %s, got %s", dstFile, detectedPath)
	}

	// Wait for file to stabilize
	if err := stab.WaitForStable(ctx, detectedPath); err != nil {
		t.Fatalf("stabilizer failed: %v", err)
	}
	t.Log("file stabilized")

	// Extract metadata
	meta, err := metadata.ExtractM4A(detectedPath)
	if err != nil {
		t.Fatalf("metadata extraction failed: %v", err)
	}

	// Verify metadata
	timeDiff := meta.CreationTime.Sub(creationTime)
	if timeDiff < 0 {
		timeDiff = -timeDiff
	}
	if timeDiff > time.Second {
		t.Errorf("creation time mismatch: expected ~%v, got %v", creationTime, meta.CreationTime)
	}

	expectedDuration := time.Duration(durationSeconds) * time.Second
	if meta.Duration != expectedDuration {
		t.Errorf("duration mismatch: expected %v, got %v", expectedDuration, meta.Duration)
	}

	t.Logf("metadata extracted: creation=%v, duration=%v", meta.CreationTime, meta.Duration)
}

// TestFullFlow_MultipleFiles tests processing multiple files in sequence
func TestFullFlow_MultipleFiles(t *testing.T) {
	watchDir := t.TempDir()
	srcDir := t.TempDir()

	w, err := watcher.NewInotifyWatcher()
	if err != nil {
		t.Fatalf("failed to create watcher: %v", err)
	}
	defer w.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	events, err := w.Watch(ctx, watchDir, []string{"*.m4a"})
	if err != nil {
		t.Fatalf("failed to start watch: %v", err)
	}

	stab := stabilizer.NewPollStabilizer(30*time.Millisecond, 3)

	time.Sleep(100 * time.Millisecond)

	// Process 3 files
	testFiles := []struct {
		name     string
		duration uint32
	}{
		{"note1.m4a", 30},
		{"note2.m4a", 60},
		{"note3.m4a", 45},
	}

	for _, tf := range testFiles {
		t.Run(tf.name, func(t *testing.T) {
			srcFile := filepath.Join(srcDir, tf.name)
			dstFile := filepath.Join(watchDir, tf.name)

			if err := createTestM4A(srcFile, time.Now().UTC(), tf.duration); err != nil {
				t.Fatalf("failed to create %s: %v", tf.name, err)
			}

			if err := os.Rename(srcFile, dstFile); err != nil {
				t.Fatalf("failed to move %s: %v", tf.name, err)
			}

			select {
			case event := <-events:
				if filepath.Base(event.Path) != tf.name {
					t.Errorf("wrong file detected: expected %s, got %s", tf.name, filepath.Base(event.Path))
				}

				if err := stab.WaitForStable(ctx, event.Path); err != nil {
					t.Fatalf("stabilizer failed for %s: %v", tf.name, err)
				}

				meta, err := metadata.ExtractM4A(event.Path)
				if err != nil {
					t.Fatalf("metadata extraction failed for %s: %v", tf.name, err)
				}

				expectedDuration := time.Duration(tf.duration) * time.Second
				if meta.Duration != expectedDuration {
					t.Errorf("duration mismatch for %s: expected %v, got %v", tf.name, expectedDuration, meta.Duration)
				}

			case <-ctx.Done():
				t.Fatalf("timeout waiting for %s", tf.name)
			}
		})
	}
}

// TestFullFlow_SlowWrite tests handling files that are written slowly
func TestFullFlow_SlowWrite(t *testing.T) {
	watchDir := t.TempDir()

	w, err := watcher.NewInotifyWatcher()
	if err != nil {
		t.Fatalf("failed to create watcher: %v", err)
	}
	defer w.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	events, err := w.Watch(ctx, watchDir, []string{"*.m4a"})
	if err != nil {
		t.Fatalf("failed to start watch: %v", err)
	}

	// Use longer stabilization for slow writes
	stab := stabilizer.NewPollStabilizer(50*time.Millisecond, 4)

	time.Sleep(100 * time.Millisecond)

	// Create file incrementally (simulating slow network transfer)
	dstFile := filepath.Join(watchDir, "slow.m4a")
	go func() {
		// Write file in chunks with delays
		f, err := os.Create(dstFile)
		if err != nil {
			return
		}
		defer f.Close()

		// Write ftyp box first
		ftyp := []byte{
			0x00, 0x00, 0x00, 0x14,
			'f', 't', 'y', 'p',
			'M', '4', 'A', ' ',
			0x00, 0x00, 0x00, 0x00,
			'M', '4', 'A', ' ',
		}
		f.Write(ftyp)
		f.Sync()
		time.Sleep(60 * time.Millisecond)

		// Write moov header
		moovHeader := []byte{
			0x00, 0x00, 0x00, 0x7C, // moov size
			'm', 'o', 'o', 'v',
		}
		f.Write(moovHeader)
		f.Sync()
		time.Sleep(60 * time.Millisecond)

		// Write mvhd
		mvhd := make([]byte, 116)
		copy(mvhd[0:4], []byte{0x00, 0x00, 0x00, 0x74}) // size
		copy(mvhd[4:8], []byte{'m', 'v', 'h', 'd'})
		// rest is zeros which is a valid minimal mvhd
		f.Write(mvhd)
		f.Sync()
	}()

	// Wait for detection
	select {
	case event := <-events:
		t.Logf("detected slow write file: %s", event.Path)

		// Stabilizer should wait for all writes to complete
		if err := stab.WaitForStable(ctx, event.Path); err != nil {
			t.Fatalf("stabilizer failed: %v", err)
		}

		// File should be complete now
		info, err := os.Stat(event.Path)
		if err != nil {
			t.Fatalf("failed to stat file: %v", err)
		}

		// ftyp (20) + moov header (8) + mvhd (116) = 144 bytes
		expectedSize := int64(144)
		if info.Size() != expectedSize {
			t.Errorf("unexpected file size: expected %d, got %d", expectedSize, info.Size())
		}

	case <-ctx.Done():
		t.Fatal("timeout waiting for slow write detection")
	}
}

// createTestM4A creates a minimal valid M4A file for testing.
func createTestM4A(path string, creationTime time.Time, durationSeconds uint32) error {
	return createM4AWithMetadata(path, creationTime, durationSeconds)
}

// createM4AWithMetadata is a helper that creates a valid M4A file.
func createM4AWithMetadata(path string, creationTime time.Time, durationSeconds uint32) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	// ftyp box
	ftyp := []byte{
		0x00, 0x00, 0x00, 0x14,
		'f', 't', 'y', 'p',
		'M', '4', 'A', ' ',
		0x00, 0x00, 0x00, 0x00,
		'M', '4', 'A', ' ',
	}
	if _, err := f.Write(ftyp); err != nil {
		return err
	}

	// Mac epoch conversion
	macEpoch := time.Date(1904, 1, 1, 0, 0, 0, 0, time.UTC)
	macTime := uint32(creationTime.Sub(macEpoch).Seconds())

	// mvhd box data
	mvhdData := make([]byte, 108)
	putUint32BE(mvhdData[4:8], macTime)               // creation time
	putUint32BE(mvhdData[8:12], macTime)              // modification time
	putUint32BE(mvhdData[12:16], 1000)                // timescale
	putUint32BE(mvhdData[16:20], durationSeconds*1000) // duration

	// mvhd box
	mvhdBox := make([]byte, 8+108)
	putUint32BE(mvhdBox[0:4], 116)
	copy(mvhdBox[4:8], []byte("mvhd"))
	copy(mvhdBox[8:], mvhdData)

	// moov box
	moovSize := uint32(8 + len(mvhdBox))
	moovHeader := make([]byte, 8)
	putUint32BE(moovHeader[0:4], moovSize)
	copy(moovHeader[4:8], []byte("moov"))

	if _, err := f.Write(moovHeader); err != nil {
		return err
	}
	if _, err := f.Write(mvhdBox); err != nil {
		return err
	}

	return nil
}

func putUint32BE(b []byte, v uint32) {
	b[0] = byte(v >> 24)
	b[1] = byte(v >> 16)
	b[2] = byte(v >> 8)
	b[3] = byte(v)
}
