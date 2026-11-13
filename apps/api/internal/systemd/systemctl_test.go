package systemd

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestUnitFileInstalled(t *testing.T) {
	fakeSystemctl(t, "#!/bin/sh\necho 'podman.socket enabled enabled'\nexit 0\n")

	installed, err := UnitFileInstalled(context.Background(), "podman.socket")
	if err != nil {
		t.Fatalf("UnitFileInstalled returned an error: %v", err)
	}
	if !installed {
		t.Error("expected a systemd that lists the unit file to report it installed")
	}
}

func TestUnitFileInstalledReportsAMissingUnit(t *testing.T) {
	fakeSystemctl(t, "#!/bin/sh\nexit 0\n")

	installed, err := UnitFileInstalled(context.Background(), "podman.socket")
	if err != nil {
		t.Fatalf("UnitFileInstalled returned an error: %v", err)
	}
	if installed {
		t.Error("expected empty output to report the unit file missing")
	}
}

func TestUnitFileInstalledSurfacesASystemctlFailure(t *testing.T) {
	fakeSystemctl(t, "#!/bin/sh\necho 'Failed to connect to bus' >&2\nexit 1\n")

	if _, err := UnitFileInstalled(context.Background(), "podman.socket"); err == nil {
		t.Error("expected a failing systemctl to return an error rather than an answer")
	}
}

// fakeSystemctl puts a stub `systemctl` first on PATH, so a query's decision can be asserted
// without depending on the systemd the machine running the tests happens to have.
func fakeSystemctl(t *testing.T, script string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "systemctl"), []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}
