package stabilizer

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestPollStabilizer_WaitsForStableFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "growing.txt")

	// Create initial file
	f, err := os.Create(testFile)
	if err != nil {
		t.Fatalf("failed to create file: %v", err)
	}
	if _, err := f.WriteString("initial"); err != nil {
		t.Fatalf("failed to write initial data: %v", err)
	}
	f.Close()

	stabilizer := NewPollStabilizer(50*time.Millisecond, 3)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Simulate file still being written
	var wg sync.WaitGroup
	wg.Add(1)
	writeDone := make(chan struct{})
	go func() {
		defer wg.Done()
		defer close(writeDone)
		// Append to file a few times
		for i := 0; i < 3; i++ {
			time.Sleep(30 * time.Millisecond)
			f, err := os.OpenFile(testFile, os.O_APPEND|os.O_WRONLY, 0644)
			if err != nil {
				return
			}
			f.WriteString(" more data")
			f.Close()
		}
	}()

	start := time.Now()
	err = stabilizer.WaitForStable(ctx, testFile)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("WaitForStable failed: %v", err)
	}

	// Should have waited at least interval * checks after writes stopped
	// Plus the time for writes to complete (~90ms)
	wg.Wait()
	if elapsed < 150*time.Millisecond {
		t.Errorf("stabilizer returned too quickly: %v", elapsed)
	}
}

func TestPollStabilizer_ImmediateStable(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "stable.txt")

	// Create a file that won't change
	if err := os.WriteFile(testFile, []byte("stable content"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	stabilizer := NewPollStabilizer(10*time.Millisecond, 3)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	start := time.Now()
	err := stabilizer.WaitForStable(ctx, testFile)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("WaitForStable failed: %v", err)
	}

	// Should wait interval * checks (30ms minimum)
	if elapsed < 30*time.Millisecond {
		t.Errorf("stabilizer returned too quickly: %v", elapsed)
	}
	if elapsed > time.Second {
		t.Errorf("stabilizer took too long: %v", elapsed)
	}
}

func TestPollStabilizer_ContextCancellation(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	// Create a file that we'll keep modifying
	f, err := os.Create(testFile)
	if err != nil {
		t.Fatalf("failed to create file: %v", err)
	}
	f.Close()

	stabilizer := NewPollStabilizer(100*time.Millisecond, 10)

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()

	// Keep modifying the file
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				f, err := os.OpenFile(testFile, os.O_APPEND|os.O_WRONLY, 0644)
				if err != nil {
					return
				}
				f.WriteString("x")
				f.Close()
				time.Sleep(50 * time.Millisecond)
			}
		}
	}()

	err = stabilizer.WaitForStable(ctx, testFile)
	if err != context.DeadlineExceeded {
		t.Errorf("expected DeadlineExceeded, got: %v", err)
	}
}

func TestPollStabilizer_FileNotFound(t *testing.T) {
	stabilizer := NewPollStabilizer(10*time.Millisecond, 3)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := stabilizer.WaitForStable(ctx, "/nonexistent/file.txt")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestPollStabilizer_ResetOnSizeChange(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "changing.txt")

	// Create initial file
	if err := os.WriteFile(testFile, []byte("initial"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	stabilizer := NewPollStabilizer(20*time.Millisecond, 5)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// After 2 checks (~40ms), change the file
	go func() {
		time.Sleep(40 * time.Millisecond)
		f, err := os.OpenFile(testFile, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return
		}
		f.WriteString(" appended")
		f.Close()
	}()

	start := time.Now()
	err := stabilizer.WaitForStable(ctx, testFile)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("WaitForStable failed: %v", err)
	}

	// Should have taken at least:
	// ~40ms until file change + 5 checks * 20ms = 140ms minimum
	if elapsed < 130*time.Millisecond {
		t.Errorf("stabilizer should have reset counter on size change, elapsed: %v", elapsed)
	}
}
