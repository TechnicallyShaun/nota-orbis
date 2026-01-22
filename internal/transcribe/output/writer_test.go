package output

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TechnicallyShaun/nota-orbis/internal/transcribe"
)

func TestWriter_Write_PlainMarkdown(t *testing.T) {
	tmpDir := t.TempDir()
	writer := NewWriter()

	ts := time.Date(2024, 3, 15, 14, 30, 0, 0, time.UTC)
	opts := transcribe.OutputOptions{
		OutputDir:  tmpDir,
		SourceFile: "/path/to/audio.m4a",
		Timestamp:  ts,
	}

	path, err := writer.Write(context.Background(), "Hello, this is a test transcription.", opts)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Verify filename format
	expectedFilename := "2024-03-15-1430-voice-note.md"
	if filepath.Base(path) != expectedFilename {
		t.Errorf("unexpected filename: got %s, want %s", filepath.Base(path), expectedFilename)
	}

	// Verify file content
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "# Voice Note") {
		t.Error("missing header in output")
	}
	if !strings.Contains(contentStr, "**Date:** 2024-03-15 14:30") {
		t.Error("missing date in output")
	}
	if !strings.Contains(contentStr, "**Source:** audio.m4a") {
		t.Error("missing source in output")
	}
	if !strings.Contains(contentStr, "## Transcription") {
		t.Error("missing transcription header in output")
	}
	if !strings.Contains(contentStr, "Hello, this is a test transcription.") {
		t.Error("missing transcription text in output")
	}
}

func TestWriter_Write_WithTemplate(t *testing.T) {
	tmpDir := t.TempDir()
	writer := NewWriter()

	// Create a template file
	templatePath := filepath.Join(tmpDir, "template.md")
	templateContent := `---
tags: voice-note
---

# My Voice Note

`
	if err := os.WriteFile(templatePath, []byte(templateContent), 0644); err != nil {
		t.Fatalf("failed to create template: %v", err)
	}

	ts := time.Date(2024, 3, 15, 14, 30, 0, 0, time.UTC)
	outputDir := filepath.Join(tmpDir, "output")
	opts := transcribe.OutputOptions{
		OutputDir:    outputDir,
		TemplatePath: templatePath,
		Timestamp:    ts,
	}

	path, err := writer.Write(context.Background(), "Transcribed content here.", opts)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Verify file content includes template and transcription
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "tags: voice-note") {
		t.Error("missing template frontmatter in output")
	}
	if !strings.Contains(contentStr, "# My Voice Note") {
		t.Error("missing template header in output")
	}
	if !strings.Contains(contentStr, "Transcribed content here.") {
		t.Error("missing transcription text in output")
	}
}

func TestWriter_Write_CollisionHandling(t *testing.T) {
	tmpDir := t.TempDir()
	writer := NewWriter()

	ts := time.Date(2024, 3, 15, 14, 30, 0, 0, time.UTC)
	opts := transcribe.OutputOptions{
		OutputDir: tmpDir,
		Timestamp: ts,
	}

	// Create first file
	path1, err := writer.Write(context.Background(), "First transcription.", opts)
	if err != nil {
		t.Fatalf("Write 1 failed: %v", err)
	}
	if filepath.Base(path1) != "2024-03-15-1430-voice-note.md" {
		t.Errorf("unexpected first filename: %s", filepath.Base(path1))
	}

	// Create second file with same timestamp - should get -2 suffix
	path2, err := writer.Write(context.Background(), "Second transcription.", opts)
	if err != nil {
		t.Fatalf("Write 2 failed: %v", err)
	}
	if filepath.Base(path2) != "2024-03-15-1430-voice-note-2.md" {
		t.Errorf("unexpected second filename: %s", filepath.Base(path2))
	}

	// Create third file - should get -3 suffix
	path3, err := writer.Write(context.Background(), "Third transcription.", opts)
	if err != nil {
		t.Fatalf("Write 3 failed: %v", err)
	}
	if filepath.Base(path3) != "2024-03-15-1430-voice-note-3.md" {
		t.Errorf("unexpected third filename: %s", filepath.Base(path3))
	}

	// Verify all files exist with correct content
	for _, tc := range []struct {
		path    string
		content string
	}{
		{path1, "First transcription."},
		{path2, "Second transcription."},
		{path3, "Third transcription."},
	} {
		data, err := os.ReadFile(tc.path)
		if err != nil {
			t.Errorf("failed to read %s: %v", tc.path, err)
			continue
		}
		if !strings.Contains(string(data), tc.content) {
			t.Errorf("file %s missing expected content", tc.path)
		}
	}
}

func TestWriter_Write_CreatesOutputDir(t *testing.T) {
	tmpDir := t.TempDir()
	writer := NewWriter()

	// Use a nested directory that doesn't exist
	outputDir := filepath.Join(tmpDir, "nested", "output", "dir")
	opts := transcribe.OutputOptions{
		OutputDir: outputDir,
		Timestamp: time.Now(),
	}

	path, err := writer.Write(context.Background(), "Test.", opts)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Verify directory was created
	info, err := os.Stat(outputDir)
	if err != nil {
		t.Fatalf("output directory not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("output path is not a directory")
	}

	// Verify file exists
	if _, err := os.Stat(path); err != nil {
		t.Errorf("output file not found: %v", err)
	}
}

func TestWriter_Write_RequiresOutputDir(t *testing.T) {
	writer := NewWriter()

	opts := transcribe.OutputOptions{
		Timestamp: time.Now(),
	}

	_, err := writer.Write(context.Background(), "Test.", opts)
	if err == nil {
		t.Fatal("expected error for missing OutputDir")
	}
	if !strings.Contains(err.Error(), "output directory is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestWriter_Write_TemplateNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	writer := NewWriter()

	opts := transcribe.OutputOptions{
		OutputDir:    tmpDir,
		TemplatePath: "/nonexistent/template.md",
		Timestamp:    time.Now(),
	}

	_, err := writer.Write(context.Background(), "Test.", opts)
	if err == nil {
		t.Fatal("expected error for missing template")
	}
	if !strings.Contains(err.Error(), "failed to read template") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestWriter_Write_ContextCancellation(t *testing.T) {
	writer := NewWriter()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	opts := transcribe.OutputOptions{
		OutputDir: t.TempDir(),
		Timestamp: time.Now(),
	}

	_, err := writer.Write(ctx, "Test.", opts)
	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got: %v", err)
	}
}

func TestWriter_Write_DefaultTimestamp(t *testing.T) {
	tmpDir := t.TempDir()
	writer := NewWriter()

	opts := transcribe.OutputOptions{
		OutputDir: tmpDir,
		// Timestamp is zero, should use current time
	}

	before := time.Now()
	path, err := writer.Write(context.Background(), "Test.", opts)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	after := time.Now()

	// Extract date from filename
	filename := filepath.Base(path)
	// Format: YYYY-MM-DD-HHmm-voice-note.md
	parts := strings.Split(filename, "-voice-note")
	if len(parts) != 2 {
		t.Fatalf("unexpected filename format: %s", filename)
	}

	dateStr := parts[0]
	parsed, err := time.Parse("2006-01-02-1504", dateStr)
	if err != nil {
		t.Fatalf("failed to parse date from filename: %v", err)
	}

	// Verify the timestamp is within the expected range
	if parsed.Before(before.Truncate(time.Minute)) || parsed.After(after.Add(time.Minute)) {
		t.Errorf("filename timestamp %v not in expected range [%v, %v]", parsed, before, after)
	}
}

func TestWriter_Write_TemplateWithoutTrailingNewline(t *testing.T) {
	tmpDir := t.TempDir()
	writer := NewWriter()

	// Create a template file without trailing newline
	templatePath := filepath.Join(tmpDir, "template.md")
	templateContent := "# No trailing newline"
	if err := os.WriteFile(templatePath, []byte(templateContent), 0644); err != nil {
		t.Fatalf("failed to create template: %v", err)
	}

	outputDir := filepath.Join(tmpDir, "output")
	opts := transcribe.OutputOptions{
		OutputDir:    outputDir,
		TemplatePath: templatePath,
		Timestamp:    time.Now(),
	}

	path, err := writer.Write(context.Background(), "Transcribed text.", opts)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read output: %v", err)
	}

	// Should have proper newlines between template and transcription
	expected := "# No trailing newline\n\nTranscribed text.\n"
	if string(content) != expected {
		t.Errorf("content mismatch:\ngot:\n%q\nwant:\n%q", string(content), expected)
	}
}

func TestWriter_Write_NoSourceFile(t *testing.T) {
	tmpDir := t.TempDir()
	writer := NewWriter()

	opts := transcribe.OutputOptions{
		OutputDir: tmpDir,
		Timestamp: time.Date(2024, 3, 15, 14, 30, 0, 0, time.UTC),
		// No SourceFile set
	}

	path, err := writer.Write(context.Background(), "Test content.", opts)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read output: %v", err)
	}

	// Should not contain Source line
	if strings.Contains(string(content), "**Source:**") {
		t.Error("output should not contain Source when SourceFile is empty")
	}
}
