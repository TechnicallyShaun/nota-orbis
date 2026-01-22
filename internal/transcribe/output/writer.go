// Package output provides file writing functionality for transcription results.
package output

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/TechnicallyShaun/nota-orbis/internal/transcribe"
)

// Compile-time check that Writer implements transcribe.OutputWriter.
var _ transcribe.OutputWriter = (*Writer)(nil)

// Writer implements transcribe.OutputWriter for saving transcriptions to markdown files.
type Writer struct{}

// NewWriter creates a new OutputWriter.
func NewWriter() *Writer {
	return &Writer{}
}

// Write saves the transcription text and returns the path to the created file.
// If opts.TemplatePath is set, the template is read and transcription is appended.
// Otherwise, plain markdown with the transcription is written.
func (w *Writer) Write(ctx context.Context, text string, opts transcribe.OutputOptions) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	if opts.OutputDir == "" {
		return "", fmt.Errorf("output directory is required")
	}

	// Ensure output directory exists
	if err := os.MkdirAll(opts.OutputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	// Generate filename with collision handling
	filename, err := w.generateFilename(opts)
	if err != nil {
		return "", fmt.Errorf("failed to generate filename: %w", err)
	}

	outputPath := filepath.Join(opts.OutputDir, filename)

	// Generate content
	content, err := w.generateContent(text, opts)
	if err != nil {
		return "", fmt.Errorf("failed to generate content: %w", err)
	}

	// Write to file
	if err := os.WriteFile(outputPath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to write output file: %w", err)
	}

	return outputPath, nil
}

// generateFilename creates a filename in the format YYYY-MM-DD-HHmm-voice-note.md
// with collision handling (-2, -3, etc.).
func (w *Writer) generateFilename(opts transcribe.OutputOptions) (string, error) {
	ts := opts.Timestamp
	if ts.IsZero() {
		ts = time.Now()
	}

	// Format: YYYY-MM-DD-HHmm-voice-note.md
	baseName := ts.Format("2006-01-02-1504") + "-voice-note"
	ext := ".md"

	// Check for collision and add suffix if needed
	filename := baseName + ext
	candidate := filepath.Join(opts.OutputDir, filename)

	if _, err := os.Stat(candidate); os.IsNotExist(err) {
		return filename, nil
	}

	// Handle collision with -2, -3, etc.
	for i := 2; i <= 1000; i++ {
		filename = fmt.Sprintf("%s-%d%s", baseName, i, ext)
		candidate = filepath.Join(opts.OutputDir, filename)
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return filename, nil
		}
	}

	return "", fmt.Errorf("too many files with same timestamp")
}

// generateContent creates the file content, optionally using a template.
func (w *Writer) generateContent(text string, opts transcribe.OutputOptions) (string, error) {
	if opts.TemplatePath != "" {
		return w.generateFromTemplate(text, opts)
	}
	return w.generatePlainMarkdown(text, opts), nil
}

// generateFromTemplate reads the template file and appends the transcription.
func (w *Writer) generateFromTemplate(text string, opts transcribe.OutputOptions) (string, error) {
	templateContent, err := os.ReadFile(opts.TemplatePath)
	if err != nil {
		return "", fmt.Errorf("failed to read template: %w", err)
	}

	var sb strings.Builder
	sb.Write(templateContent)

	// Ensure there's a newline before appending transcription
	if len(templateContent) > 0 && templateContent[len(templateContent)-1] != '\n' {
		sb.WriteString("\n")
	}
	sb.WriteString("\n")
	sb.WriteString(text)
	sb.WriteString("\n")

	return sb.String(), nil
}

// generatePlainMarkdown creates a simple markdown document with the transcription.
func (w *Writer) generatePlainMarkdown(text string, opts transcribe.OutputOptions) string {
	ts := opts.Timestamp
	if ts.IsZero() {
		ts = time.Now()
	}

	var sb strings.Builder
	sb.WriteString("# Voice Note\n\n")
	sb.WriteString(fmt.Sprintf("**Date:** %s\n\n", ts.Format("2006-01-02 15:04")))

	if opts.SourceFile != "" {
		sb.WriteString(fmt.Sprintf("**Source:** %s\n\n", filepath.Base(opts.SourceFile)))
	}

	sb.WriteString("## Transcription\n\n")
	sb.WriteString(text)
	sb.WriteString("\n")

	return sb.String()
}
