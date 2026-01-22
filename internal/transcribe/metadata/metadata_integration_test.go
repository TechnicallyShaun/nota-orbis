package metadata

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestExtractM4A_ValidFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.m4a")

	// Create a test M4A with known metadata
	creationTime := time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC)
	durationSeconds := uint32(120) // 2 minutes

	if err := createTestM4A(testFile, creationTime, durationSeconds); err != nil {
		t.Fatalf("failed to create test M4A: %v", err)
	}

	meta, err := ExtractM4A(testFile)
	if err != nil {
		t.Fatalf("ExtractM4A failed: %v", err)
	}

	// Check creation time (allow some tolerance for conversion)
	timeDiff := meta.CreationTime.Sub(creationTime)
	if timeDiff < 0 {
		timeDiff = -timeDiff
	}
	if timeDiff > time.Second {
		t.Errorf("creation time mismatch: expected ~%v, got %v", creationTime, meta.CreationTime)
	}

	// Check duration
	expectedDuration := time.Duration(durationSeconds) * time.Second
	if meta.Duration != expectedDuration {
		t.Errorf("duration mismatch: expected %v, got %v", expectedDuration, meta.Duration)
	}
}

func TestExtractM4A_InvalidFormat(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "invalid.m4a")

	if err := createInvalidM4A(testFile); err != nil {
		t.Fatalf("failed to create invalid M4A: %v", err)
	}

	_, err := ExtractM4A(testFile)
	if err != ErrInvalidFormat {
		t.Errorf("expected ErrInvalidFormat, got: %v", err)
	}
}

func TestExtractM4A_NonexistentFile(t *testing.T) {
	_, err := ExtractM4A("/nonexistent/file.m4a")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestExtractM4A_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "empty.m4a")

	// Create an empty file
	f, err := createEmptyFile(testFile)
	if err != nil {
		t.Fatalf("failed to create empty file: %v", err)
	}
	f.Close()

	_, err = ExtractM4A(testFile)
	if err == nil {
		t.Error("expected error for empty file")
	}
}

func TestExtractM4A_DifferentDurations(t *testing.T) {
	tests := []struct {
		name     string
		duration uint32
	}{
		{"short", 10},       // 10 seconds
		{"medium", 300},     // 5 minutes
		{"long", 3600},      // 1 hour
		{"very_long", 7200}, // 2 hours
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			testFile := filepath.Join(tmpDir, "test.m4a")

			creationTime := time.Now().UTC().Truncate(time.Second)
			if err := createTestM4A(testFile, creationTime, tt.duration); err != nil {
				t.Fatalf("failed to create test M4A: %v", err)
			}

			meta, err := ExtractM4A(testFile)
			if err != nil {
				t.Fatalf("ExtractM4A failed: %v", err)
			}

			expectedDuration := time.Duration(tt.duration) * time.Second
			if meta.Duration != expectedDuration {
				t.Errorf("duration mismatch: expected %v, got %v", expectedDuration, meta.Duration)
			}
		})
	}
}

func TestExtractM4A_HistoricalDates(t *testing.T) {
	tests := []struct {
		name string
		time time.Time
	}{
		{"recent", time.Date(2026, 1, 20, 14, 30, 0, 0, time.UTC)},
		{"year_ago", time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)},
		{"decade_ago", time.Date(2016, 6, 1, 8, 0, 0, 0, time.UTC)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			testFile := filepath.Join(tmpDir, "test.m4a")

			if err := createTestM4A(testFile, tt.time, 60); err != nil {
				t.Fatalf("failed to create test M4A: %v", err)
			}

			meta, err := ExtractM4A(testFile)
			if err != nil {
				t.Fatalf("ExtractM4A failed: %v", err)
			}

			timeDiff := meta.CreationTime.Sub(tt.time)
			if timeDiff < 0 {
				timeDiff = -timeDiff
			}
			if timeDiff > time.Second {
				t.Errorf("creation time mismatch: expected ~%v, got %v (diff: %v)", tt.time, meta.CreationTime, timeDiff)
			}
		})
	}
}

func createEmptyFile(path string) (*os.File, error) {
	return os.Create(path)
}
