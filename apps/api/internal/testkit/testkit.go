// Package testkit holds the test helpers the cli, engine and workload tests share: stub commands
// on PATH that record how they were called, and unit file fixtures. It is imported by tests only.
package testkit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FacileStudio/gare/apps/api/internal/systemd"
)

// StubPodman puts a recording `podman` first on PATH and returns the file it logs each invocation
// to, so the commands a teardown runs can be asserted without depending on a real podman.
func StubPodman(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	logPath := filepath.Join(dir, "podman.log")
	script := "#!/bin/sh\nprintf '%s\\n' \"$*\" >> \"$GARE_TEST_PODMAN_LOG\"\nexit 0\n"
	if err := os.WriteFile(filepath.Join(dir, "podman"), []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("GARE_TEST_PODMAN_LOG", logPath)
	return logPath
}

// RecordedCalls returns the lines one of the path stubs logged, nil when it was never invoked.
func RecordedCalls(t *testing.T, logPath string) []string {
	t.Helper()
	data, err := os.ReadFile(logPath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		t.Fatalf("failed to read the call log: %v", err)
	}
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "\n")
}

// WriteUnitFile writes a unit file under the named application's unit path and returns that path.
func WriteUnitFile(t *testing.T, name, content string) string {
	t.Helper()
	unitPath := systemd.GetUnitPath(name)
	if err := os.MkdirAll(filepath.Dir(unitPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(unitPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return unitPath
}

// AssertGone fails the test when path still exists.
func AssertGone(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Lstat(path); !os.IsNotExist(err) {
		t.Errorf("expected %s to be removed, got %v", path, err)
	}
}
