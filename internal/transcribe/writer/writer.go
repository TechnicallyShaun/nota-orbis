// Package writer provides output writing capabilities for the transcription service.
package writer

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// OutputOptions configures output writing.
type OutputOptions struct {
	OutputDir    string
	TemplatePath string
	SourceFile   string
	Timestamp    time.Time
}

// OutputWriter saves transcriptions to the vault.
type OutputWriter interface {
	Write(ctx context.Context, text string, opts OutputOptions) (string, error)
}

// SimpleWriter implements OutputWriter with basic file writing.
type SimpleWriter struct{}

// NewSimpleWriter creates a new simple output writer.
func NewSimpleWriter() *SimpleWriter {
	return &SimpleWriter{}
}

// Write saves the transcription text to a markdown file.
// The file is named based on the source audio file with a .md extension.
func (w *SimpleWriter) Write(ctx context.Context, text string, opts OutputOptions) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	// Ensure output directory exists
	if err := os.MkdirAll(opts.OutputDir, 0755); err != nil {
		return "", fmt.Errorf("create output directory: %w", err)
	}

	// Generate output filename from source file
	baseName := filepath.Base(opts.SourceFile)
	ext := filepath.Ext(baseName)
	nameWithoutExt := strings.TrimSuffix(baseName, ext)

	// Add timestamp to filename for uniqueness
	timestamp := opts.Timestamp
	if timestamp.IsZero() {
		timestamp = time.Now()
	}
	dateStr := timestamp.Format("2006-01-02-150405")
	outputName := fmt.Sprintf("%s-%s.md", nameWithoutExt, dateStr)
	outputPath := filepath.Join(opts.OutputDir, outputName)

	// Write the transcription
	content := formatTranscription(text, opts)
	if err := os.WriteFile(outputPath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("write transcription file: %w", err)
	}

	return outputPath, nil
}

// formatTranscription formats the transcription text with metadata.
func formatTranscription(text string, opts OutputOptions) string {
	var sb strings.Builder

	// YAML frontmatter
	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("source: %s\n", filepath.Base(opts.SourceFile)))
	if !opts.Timestamp.IsZero() {
		sb.WriteString(fmt.Sprintf("transcribed: %s\n", opts.Timestamp.Format(time.RFC3339)))
	}
	sb.WriteString("type: transcription\n")
	sb.WriteString("---\n\n")

	// Transcription content
	sb.WriteString("# Transcription\n\n")
	sb.WriteString(text)
	sb.WriteString("\n")

	return sb.String()
}
