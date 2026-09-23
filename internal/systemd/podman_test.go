package systemd

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckPodmanWorkloadsAcceptsASupportedPodman(t *testing.T) {
	fakePodman(t, "#!/bin/sh\necho 'Usage: podman kube play'\nexit 0\n")

	if err := CheckPodmanWorkloads(context.Background()); err != nil {
		t.Errorf("expected a podman that accepts the flags to pass the probe, got %v", err)
	}
}

func TestCheckPodmanWorkloadsNamesTheVersionFloor(t *testing.T) {
	fakePodman(t, "#!/bin/sh\necho 'Error: unknown flag: --service-container' >&2\nexit 125\n")

	err := CheckPodmanWorkloads(context.Background())
	if err == nil {
		t.Fatal("expected a podman that rejects the flags to fail the probe")
	}
	if !strings.Contains(err.Error(), "unknown flag") {
		t.Errorf("the error must carry podman's own diagnostic, got %v", err)
	}
	if !strings.Contains(err.Error(), "5.0") {
		t.Errorf("the error must name the version floor that fixes it, got %v", err)
	}
}

func TestCheckPodmanWorkloadsReportsASilentFailure(t *testing.T) {
	fakePodman(t, "#!/bin/sh\nexit 1\n")

	err := CheckPodmanWorkloads(context.Background())
	if err == nil {
		t.Fatal("expected a failing probe to return an error")
	}
	if strings.Contains(err.Error(), "():") {
		t.Errorf("an empty diagnostic must not leave a placeholder in the message, got %v", err)
	}
}

// fakePodman puts a stub `podman` first on PATH, so the probe's decision can be asserted without
// depending on whichever podman the machine running the tests happens to have installed.
func fakePodman(t *testing.T, script string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "podman"), []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}
