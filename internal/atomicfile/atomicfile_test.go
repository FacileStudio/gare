package atomicfile

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "output.txt")
	data := []byte("hello world")

	if err := WriteFile(path, data, 0644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	if string(content) != "hello world" {
		t.Errorf("got %q, want %q", string(content), "hello world")
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat failed: %v", err)
	}
	if info.Mode().Perm() != 0644 {
		t.Errorf("got perm %o, want 0644", info.Mode().Perm())
	}
}

func TestWriteFile_CreatesDir(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "sub", "nested", "output.txt")
	data := []byte("nested")

	if err := WriteFile(path, data, 0644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("file was not created")
	}
}

func TestWriteFile_TempFileCleanup(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "output.txt")

	if err := WriteFile(path, []byte("real content"), 0644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertNoTempFiles(t, tmpDir)
}

func TestWriteFile_EmptyData(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "empty.txt")

	if err := WriteFile(path, []byte{}, 0644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read: %v", err)
	}
	if len(content) != 0 {
		t.Errorf("got %d bytes, want 0", len(content))
	}
}

func TestWriteFileReplacesTheFileItRewrites(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "output.txt")

	if err := WriteFile(path, []byte("first"), 0644); err != nil {
		t.Fatalf("first write failed: %v", err)
	}
	if err := WriteFile(path, []byte("second"), 0600); err != nil {
		t.Fatalf("second write failed: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read: %v", err)
	}
	if string(content) != "second" {
		t.Errorf("content = %q, want the replacement %q", content, "second")
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat failed: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("perm = %o, want the replacement's 0600", info.Mode().Perm())
	}
	assertNoTempFiles(t, dir)
}

func TestWriteFileStagesTheTempBesideTheTarget(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "output.txt")
	t.Setenv("TMPDIR", filepath.Join(dir, "does-not-exist"))

	if err := WriteFile(path, []byte("data"), 0644); err != nil {
		t.Fatalf("expected the temp file to be staged beside the target, got %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("readdir failed: %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != "output.txt" {
		t.Errorf("directory holds %v, want only the target file", entries)
	}
}

func TestWriteFileRemovesTheTempWhenTheRenameFails(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "output.txt")
	if err := os.Mkdir(path, 0755); err != nil {
		t.Fatalf("failed to create the directory in the target's place: %v", err)
	}

	if err := WriteFile(path, []byte("data"), 0644); err == nil {
		t.Fatal("expected renaming a file over a directory to fail")
	}
	assertNoTempFiles(t, dir)
}

func TestWriteFileFailsWhenTheParentIsAFile(t *testing.T) {
	dir := t.TempDir()
	blocker := filepath.Join(dir, "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0644); err != nil {
		t.Fatalf("failed to create the blocking file: %v", err)
	}

	if err := WriteFile(filepath.Join(blocker, "nested", "output.txt"), []byte("data"), 0644); err == nil {
		t.Fatal("expected creating a directory under a file to fail")
	}
}

func TestWriteFileFailsWhenTheDirectoryIsNotWritable(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root bypasses directory permissions")
	}
	dir := t.TempDir()
	if err := os.Chmod(dir, 0500); err != nil {
		t.Fatalf("chmod failed: %v", err)
	}
	t.Cleanup(func() { os.Chmod(dir, 0700) })

	if err := WriteFile(filepath.Join(dir, "output.txt"), []byte("data"), 0644); err == nil {
		t.Fatal("expected a write into an unwritable directory to fail")
	}
	assertNoTempFiles(t, dir)
}

func TestWriteFileNeverExposesAPartialWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "output.txt")
	old, replacement := bytes.Repeat([]byte("a"), 4096), bytes.Repeat([]byte("b"), 4096)
	if err := WriteFile(path, old, 0644); err != nil {
		t.Fatalf("initial write failed: %v", err)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		for range 100 {
			content, err := os.ReadFile(path)
			if err == nil && !bytes.Equal(content, old) && !bytes.Equal(content, replacement) {
				t.Errorf("read %d bytes, neither the old nor the new content", len(content))
				return
			}
		}
	}()
	for range 100 {
		if err := WriteFile(path, replacement, 0644); err != nil {
			t.Fatalf("rewrite failed: %v", err)
		}
	}
	<-done
}

func assertNoTempFiles(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("readdir failed: %v", err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".atomic-") {
			t.Errorf("temp file %q was left behind", entry.Name())
		}
	}
}
