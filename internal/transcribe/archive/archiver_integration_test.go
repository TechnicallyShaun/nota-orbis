package archive

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestFileArchiver_ArchivesFile(t *testing.T) {
	tmpDir := t.TempDir()
	sourceDir := filepath.Join(tmpDir, "source")
	archiveDir := filepath.Join(tmpDir, "archive")

	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("failed to create source dir: %v", err)
	}

	// Create source file
	sourceFile := filepath.Join(sourceDir, "test-audio.m4a")
	content := []byte("fake audio content")
	if err := os.WriteFile(sourceFile, content, 0644); err != nil {
		t.Fatalf("failed to create source file: %v", err)
	}

	archiver := NewFileArchiver()
	ctx := context.Background()

	err := archiver.Archive(ctx, sourceFile, archiveDir)
	if err != nil {
		t.Fatalf("Archive failed: %v", err)
	}

	// Verify file was copied to archive
	archivedFile := filepath.Join(archiveDir, "test-audio.m4a")
	archivedContent, err := os.ReadFile(archivedFile)
	if err != nil {
		t.Fatalf("failed to read archived file: %v", err)
	}
	if string(archivedContent) != string(content) {
		t.Errorf("archived content mismatch: got %q, want %q", archivedContent, content)
	}

	// Verify original was deleted
	if _, err := os.Stat(sourceFile); !os.IsNotExist(err) {
		t.Error("original file should have been deleted")
	}
}

func TestFileArchiver_CreatesArchiveDir(t *testing.T) {
	tmpDir := t.TempDir()
	sourceFile := filepath.Join(tmpDir, "source.m4a")
	archiveDir := filepath.Join(tmpDir, "nested", "archive", "dir")

	// Create source file
	if err := os.WriteFile(sourceFile, []byte("content"), 0644); err != nil {
		t.Fatalf("failed to create source file: %v", err)
	}

	archiver := NewFileArchiver()
	ctx := context.Background()

	err := archiver.Archive(ctx, sourceFile, archiveDir)
	if err != nil {
		t.Fatalf("Archive failed: %v", err)
	}

	// Verify archive directory was created
	info, err := os.Stat(archiveDir)
	if err != nil {
		t.Fatalf("archive dir not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("archive path is not a directory")
	}

	// Verify file exists in archive
	archivedFile := filepath.Join(archiveDir, "source.m4a")
	if _, err := os.Stat(archivedFile); err != nil {
		t.Errorf("archived file not found: %v", err)
	}
}

func TestFileArchiver_PreservesFilename(t *testing.T) {
	tmpDir := t.TempDir()
	sourceFile := filepath.Join(tmpDir, "my-unique-filename-2026-01-22.m4a")
	archiveDir := filepath.Join(tmpDir, "archive")

	if err := os.WriteFile(sourceFile, []byte("content"), 0644); err != nil {
		t.Fatalf("failed to create source file: %v", err)
	}

	archiver := NewFileArchiver()
	ctx := context.Background()

	err := archiver.Archive(ctx, sourceFile, archiveDir)
	if err != nil {
		t.Fatalf("Archive failed: %v", err)
	}

	// Verify filename is preserved
	archivedFile := filepath.Join(archiveDir, "my-unique-filename-2026-01-22.m4a")
	if _, err := os.Stat(archivedFile); err != nil {
		t.Errorf("archived file with original name not found: %v", err)
	}
}

func TestFileArchiver_SourceNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	sourceFile := filepath.Join(tmpDir, "nonexistent.m4a")
	archiveDir := filepath.Join(tmpDir, "archive")

	archiver := NewFileArchiver()
	ctx := context.Background()

	err := archiver.Archive(ctx, sourceFile, archiveDir)
	if err != ErrSourceNotFound {
		t.Errorf("expected ErrSourceNotFound, got: %v", err)
	}
}

func TestFileArchiver_ContextCancellation(t *testing.T) {
	tmpDir := t.TempDir()
	sourceFile := filepath.Join(tmpDir, "source.m4a")
	archiveDir := filepath.Join(tmpDir, "archive")

	if err := os.WriteFile(sourceFile, []byte("content"), 0644); err != nil {
		t.Fatalf("failed to create source file: %v", err)
	}

	archiver := NewFileArchiver()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := archiver.Archive(ctx, sourceFile, archiveDir)
	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got: %v", err)
	}

	// Verify original file was not deleted
	if _, err := os.Stat(sourceFile); err != nil {
		t.Error("original file should not have been deleted on cancellation")
	}
}

func TestFileArchiver_CopyFailure_OriginalPreserved(t *testing.T) {
	tmpDir := t.TempDir()
	sourceFile := filepath.Join(tmpDir, "source.m4a")
	// Use an invalid path (file as directory) to force copy failure
	invalidArchiveDir := filepath.Join(tmpDir, "file-not-dir")

	content := []byte("important content")
	if err := os.WriteFile(sourceFile, content, 0644); err != nil {
		t.Fatalf("failed to create source file: %v", err)
	}

	// Create a file where we want a directory
	if err := os.WriteFile(invalidArchiveDir, []byte("blocking file"), 0644); err != nil {
		t.Fatalf("failed to create blocking file: %v", err)
	}

	archiver := NewFileArchiver()
	ctx := context.Background()

	err := archiver.Archive(ctx, sourceFile, invalidArchiveDir)
	if err == nil {
		t.Error("expected error when archive dir creation fails")
	}

	// Verify original file was preserved
	preservedContent, err := os.ReadFile(sourceFile)
	if err != nil {
		t.Fatalf("original file should be preserved: %v", err)
	}
	if string(preservedContent) != string(content) {
		t.Error("original file content was modified")
	}
}

func TestFileArchiver_PreservesFileMode(t *testing.T) {
	tmpDir := t.TempDir()
	sourceFile := filepath.Join(tmpDir, "source.m4a")
	archiveDir := filepath.Join(tmpDir, "archive")

	// Create source file with specific permissions
	if err := os.WriteFile(sourceFile, []byte("content"), 0600); err != nil {
		t.Fatalf("failed to create source file: %v", err)
	}

	archiver := NewFileArchiver()
	ctx := context.Background()

	err := archiver.Archive(ctx, sourceFile, archiveDir)
	if err != nil {
		t.Fatalf("Archive failed: %v", err)
	}

	// Verify archived file has same permissions
	archivedFile := filepath.Join(archiveDir, "source.m4a")
	info, err := os.Stat(archivedFile)
	if err != nil {
		t.Fatalf("failed to stat archived file: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("file mode not preserved: got %o, want %o", info.Mode().Perm(), 0600)
	}
}
