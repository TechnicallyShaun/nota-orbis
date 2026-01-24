package pidfile

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestPath(t *testing.T) {
	path, err := Path()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if path == "" {
		t.Error("expected non-empty path")
	}

	// Should end with .nota/transcribe.pid
	if filepath.Base(path) != "transcribe.pid" {
		t.Errorf("expected path to end with transcribe.pid, got: %s", path)
	}

	dir := filepath.Base(filepath.Dir(path))
	if dir != ".nota" {
		t.Errorf("expected parent directory to be .nota, got: %s", dir)
	}
}

func TestWriteAndRead(t *testing.T) {
	// Use a temp directory for testing
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	testPID := 12345

	// Write PID
	err := Write(testPID)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Read PID
	pid, err := Read()
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	if pid != testPID {
		t.Errorf("expected PID %d, got %d", testPID, pid)
	}

	// Verify file permissions
	path, _ := Path()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}

	expectedPerm := os.FileMode(0644)
	if info.Mode().Perm() != expectedPerm {
		t.Errorf("expected permissions %o, got %o", expectedPerm, info.Mode().Perm())
	}
}

func TestReadNoPIDFile(t *testing.T) {
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	_, err := Read()
	if err != ErrNoPIDFile {
		t.Errorf("expected ErrNoPIDFile, got: %v", err)
	}
}

func TestReadInvalidPID(t *testing.T) {
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// Create directory structure
	notaDir := filepath.Join(tmpDir, ".nota")
	os.MkdirAll(notaDir, 0755)

	// Write invalid content
	path := filepath.Join(notaDir, "transcribe.pid")
	os.WriteFile(path, []byte("not-a-number\n"), 0644)

	_, err := Read()
	if err != ErrInvalidPID {
		t.Errorf("expected ErrInvalidPID, got: %v", err)
	}
}

func TestReadNegativePID(t *testing.T) {
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// Create directory structure
	notaDir := filepath.Join(tmpDir, ".nota")
	os.MkdirAll(notaDir, 0755)

	// Write negative PID
	path := filepath.Join(notaDir, "transcribe.pid")
	os.WriteFile(path, []byte("-1\n"), 0644)

	_, err := Read()
	if err != ErrInvalidPID {
		t.Errorf("expected ErrInvalidPID, got: %v", err)
	}
}

func TestRemove(t *testing.T) {
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// Create a PID file
	Write(12345)

	// Remove it
	err := Remove()
	if err != nil {
		t.Fatalf("Remove failed: %v", err)
	}

	// Verify it's gone
	path, _ := Path()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("expected PID file to be removed")
	}
}

func TestRemoveNonexistent(t *testing.T) {
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// Remove should succeed even if file doesn't exist
	err := Remove()
	if err != nil {
		t.Errorf("expected no error removing nonexistent file, got: %v", err)
	}
}

func TestWriteCreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// .nota directory doesn't exist yet
	notaDir := filepath.Join(tmpDir, ".nota")
	if _, err := os.Stat(notaDir); !os.IsNotExist(err) {
		t.Fatal("expected .nota directory to not exist initially")
	}

	// Write should create it
	err := Write(12345)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	if _, err := os.Stat(notaDir); err != nil {
		t.Error("expected .nota directory to be created")
	}
}

func TestIsRunningWithCurrentProcess(t *testing.T) {
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// Write our own PID
	currentPID := os.Getpid()
	Write(currentPID)

	running, pid, err := IsRunning()
	if err != nil {
		t.Fatalf("IsRunning failed: %v", err)
	}

	if !running {
		t.Error("expected process to be running")
	}

	if pid != currentPID {
		t.Errorf("expected PID %d, got %d", currentPID, pid)
	}
}

func TestIsRunningWithNoPIDFile(t *testing.T) {
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	running, pid, err := IsRunning()
	if err != nil {
		t.Fatalf("IsRunning failed: %v", err)
	}

	if running {
		t.Error("expected running to be false with no PID file")
	}

	if pid != 0 {
		t.Errorf("expected PID 0, got %d", pid)
	}
}

func TestIsRunningWithStalePID(t *testing.T) {
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// Write a PID that's very unlikely to be running
	// Use a very high PID number that's almost certainly not in use
	stalePID := 4194300 // Near max PID on most Linux systems

	notaDir := filepath.Join(tmpDir, ".nota")
	os.MkdirAll(notaDir, 0755)
	path := filepath.Join(notaDir, "transcribe.pid")
	os.WriteFile(path, []byte(strconv.Itoa(stalePID)+"\n"), 0644)

	running, pid, err := IsRunning()
	if err != nil {
		t.Fatalf("IsRunning failed: %v", err)
	}

	if running {
		t.Skip("stale PID is unexpectedly running, skipping test")
	}

	if pid != stalePID {
		t.Errorf("expected PID %d, got %d", stalePID, pid)
	}
}

func TestCleanStaleRemovesFile(t *testing.T) {
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// Write a stale PID
	stalePID := 4194300
	notaDir := filepath.Join(tmpDir, ".nota")
	os.MkdirAll(notaDir, 0755)
	path := filepath.Join(notaDir, "transcribe.pid")
	os.WriteFile(path, []byte(strconv.Itoa(stalePID)+"\n"), 0644)

	removed, err := CleanStale()
	if err != nil {
		t.Fatalf("CleanStale failed: %v", err)
	}

	if !removed {
		t.Error("expected stale PID file to be removed")
	}

	// Verify file is gone
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("expected PID file to be removed")
	}
}

func TestCleanStaleDoesNotRemoveRunning(t *testing.T) {
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	// Write current process PID
	Write(os.Getpid())

	removed, err := CleanStale()
	if err != nil {
		t.Fatalf("CleanStale failed: %v", err)
	}

	if removed {
		t.Error("expected running process PID file to not be removed")
	}

	// Verify file still exists
	path, _ := Path()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("expected PID file to still exist")
	}
}
