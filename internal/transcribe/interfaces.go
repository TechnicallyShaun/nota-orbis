// Package transcribe provides interfaces and types for the audio transcription pipeline.
package transcribe

import (
	"context"
	"time"
)

// FileWatcher detects new files in a directory.
type FileWatcher interface {
	// Watch starts watching the specified directory for files matching the given patterns.
	// Returns a channel that emits FileEvent for each detected file.
	Watch(ctx context.Context, dir string, patterns []string) (<-chan FileEvent, error)
	// Stop stops the file watcher.
	Stop() error
}

// FileEvent represents a detected file.
type FileEvent struct {
	Path      string
	Size      int64
	Timestamp time.Time
}

// Stabilizer waits for a file to finish writing.
type Stabilizer interface {
	// WaitForStable blocks until the file at the given path has stopped changing.
	WaitForStable(ctx context.Context, path string) error
}

// TranscriptionClient sends audio to a transcription service and receives text.
type TranscriptionClient interface {
	// Transcribe sends an audio file for transcription and returns the result.
	Transcribe(ctx context.Context, audioPath string, opts TranscribeOptions) (*TranscriptionResult, error)
}

// TranscribeOptions configures the transcription request.
type TranscribeOptions struct {
	Language string
	Model    string
}

// TranscriptionResult contains the API response.
type TranscriptionResult struct {
	Text     string
	Language string
	Duration float64
}

// OutputWriter saves transcriptions to the vault.
type OutputWriter interface {
	// Write saves the transcription text and returns the path to the created file.
	Write(ctx context.Context, text string, opts OutputOptions) (string, error)
}

// OutputOptions configures output writing.
type OutputOptions struct {
	OutputDir    string
	TemplatePath string
	SourceFile   string
	Timestamp    time.Time
}

// Archiver moves processed files to an archive location.
type Archiver interface {
	// Archive moves a file from sourcePath to the archiveDir.
	Archive(ctx context.Context, sourcePath, archiveDir string) error
}

// Logger handles structured logging.
type Logger interface {
	// Info logs an informational message with optional fields.
	Info(msg string, fields ...Field)
	// Error logs an error message with the error and optional fields.
	Error(msg string, err error, fields ...Field)
	// Debug logs a debug message with optional fields.
	Debug(msg string, fields ...Field)
}

// Field represents a key-value pair for structured logging.
type Field struct {
	Key   string
	Value any
}
