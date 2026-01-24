package status

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseLogFile_Empty(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "transcribe-test.log")

	// Create empty file
	os.WriteFile(logPath, []byte(""), 0644)

	stats, err := ParseLogFile(logPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if stats.FilesProcessed != 0 {
		t.Errorf("expected 0 files processed, got %d", stats.FilesProcessed)
	}
	if stats.Errors != 0 {
		t.Errorf("expected 0 errors, got %d", stats.Errors)
	}
	if stats.LastProcessed != nil {
		t.Error("expected LastProcessed to be nil")
	}
}

func TestParseLogFile_NonExistent(t *testing.T) {
	stats, err := ParseLogFile("/nonexistent/path/transcribe.log")
	if err != nil {
		t.Fatalf("unexpected error for nonexistent file: %v", err)
	}

	if stats.FilesProcessed != 0 {
		t.Errorf("expected 0 files processed, got %d", stats.FilesProcessed)
	}
}

func TestParseLogFile_WithCompletedFiles(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "transcribe-test.log")

	logContent := `2026-01-22T10:00:00Z INFO  [service] starting transcription service watch_dir=/mnt/sync/voice-notes
2026-01-22T10:00:01Z INFO  [pipeline] processing file path=/mnt/sync/voice-notes/meeting.m4a size=1234567
2026-01-22T10:00:05Z INFO  [pipeline] transcription complete path=/mnt/sync/voice-notes/meeting.m4a language=en
2026-01-22T10:00:06Z INFO  [pipeline] file processing complete path=/mnt/sync/voice-notes/meeting.m4a output=/vault/Inbox/meeting.md elapsed=5s
2026-01-22T11:00:00Z INFO  [pipeline] processing file path=/mnt/sync/voice-notes/notes.m4a size=2345678
2026-01-22T11:00:10Z INFO  [pipeline] file processing complete path=/mnt/sync/voice-notes/notes.m4a output=/vault/Inbox/notes.md elapsed=10s
`

	os.WriteFile(logPath, []byte(logContent), 0644)

	stats, err := ParseLogFile(logPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if stats.FilesProcessed != 2 {
		t.Errorf("expected 2 files processed, got %d", stats.FilesProcessed)
	}

	if stats.LastProcessed == nil {
		t.Fatal("expected LastProcessed to be non-nil")
	}

	expectedTime, _ := time.Parse(time.RFC3339, "2026-01-22T11:00:10Z")
	if !stats.LastProcessed.Timestamp.Equal(expectedTime) {
		t.Errorf("expected timestamp %v, got %v", expectedTime, stats.LastProcessed.Timestamp)
	}

	if stats.LastProcessed.Path != "/mnt/sync/voice-notes/notes.m4a" {
		t.Errorf("expected path /mnt/sync/voice-notes/notes.m4a, got %s", stats.LastProcessed.Path)
	}

	if stats.LastProcessed.Output != "/vault/Inbox/notes.md" {
		t.Errorf("expected output /vault/Inbox/notes.md, got %s", stats.LastProcessed.Output)
	}
}

func TestParseLogFile_WithErrors(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "transcribe-test.log")

	logContent := `2026-01-22T10:00:00Z INFO  [service] starting transcription service
2026-01-22T10:00:01Z ERROR [pipeline] transcription failed error=connection refused path=/mnt/sync/voice-notes/meeting.m4a
2026-01-22T10:01:00Z INFO  [pipeline] file processing complete path=/mnt/sync/voice-notes/notes.m4a output=/vault/Inbox/notes.md elapsed=5s
2026-01-22T10:02:00Z ERROR [pipeline] failed to archive file error=permission denied path=/mnt/sync/voice-notes/audio.m4a
`

	os.WriteFile(logPath, []byte(logContent), 0644)

	stats, err := ParseLogFile(logPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if stats.FilesProcessed != 1 {
		t.Errorf("expected 1 file processed, got %d", stats.FilesProcessed)
	}

	if stats.Errors != 2 {
		t.Errorf("expected 2 errors, got %d", stats.Errors)
	}
}

func TestUnquoteIfNeeded(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`"quoted string"`, "quoted string"},
		{`unquoted`, "unquoted"},
		{`"partial`, `"partial`},
		{`partial"`, `partial"`},
		{`""`, ""},
		{`"a"`, "a"},
	}

	for _, tc := range tests {
		result := unquoteIfNeeded(tc.input)
		if result != tc.expected {
			t.Errorf("unquoteIfNeeded(%q) = %q, expected %q", tc.input, result, tc.expected)
		}
	}
}

func TestBaseName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/path/to/file.m4a", "file.m4a"},
		{"/path/to/file", "file"},
		{"file.m4a", "file.m4a"},
		{"/path/to/dir/", "dir"},
	}

	for _, tc := range tests {
		result := BaseName(tc.input)
		if result != tc.expected {
			t.Errorf("BaseName(%q) = %q, expected %q", tc.input, result, tc.expected)
		}
	}
}

func TestFormatTimestamp(t *testing.T) {
	ts, _ := time.Parse(time.RFC3339, "2026-01-22T14:30:00Z")
	result := FormatTimestamp(ts)

	// Just verify it doesn't panic and returns something reasonable
	if result == "" {
		t.Error("expected non-empty formatted timestamp")
	}
}
