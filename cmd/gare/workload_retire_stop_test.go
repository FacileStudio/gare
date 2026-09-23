package main

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/FacileStudio/gare/internal/systemd"
)

// TestWorkloadTypeChangeStopsTheOutgoingUnit pins the handover a workload type change runs: the
// outgoing unit is stopped, and by the time systemctl is asked to stop it the unit file on disk
// already holds the replacement. That order is what makes systemd still serve the definition it
// loaded and run the outgoing unit's own ExecStop, so a compose stack goes down through
// `podman compose down` instead of a teardown naming a workload the replacement never started.
func TestWorkloadTypeChangeStopsTheOutgoingUnit(t *testing.T) {
	for name, tc := range workloadWriterCases() {
		t.Run(name, func(t *testing.T) {
			want := []string{"--user stop myapp.service", tc.description}
			if got := driveWorkloadChange(t, tc); !slices.Equal(got, want) {
				t.Errorf("systemctl invocations = %v, want %v", got, want)
			}
		})
	}
}

// TestRedeployingTheSameWorkloadTypeStopsNothing keeps that stop on a workload type change alone: a
// redeploy of the same type restarts the workload gare already supervises, so a stop would take a
// healthy application down for nothing.
func TestRedeployingTheSameWorkloadTypeStopsNothing(t *testing.T) {
	got := driveWorkloadChange(t, workloadWriterCase{
		previous:    systemd.KubeUnitDescription("myapp"),
		description: systemd.KubeUnitDescription("myapp"),
		write:       writeContainerUnit,
	})
	if len(got) != 0 {
		t.Errorf("systemctl invocations = %v, want none for a redeploy of the same workload type", got)
	}
}

// driveWorkloadChange writes one workload type's unit over an outgoing unit of another type and
// returns every systemctl invocation the handover made, each stop followed by the description the
// unit file carried at that moment.
func driveWorkloadChange(t *testing.T, tc workloadWriterCase) []string {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	stubPodman(t)
	logPath := stubSystemctl(t, "myapp")
	appDir := t.TempDir()
	writeUnitFile(t, "myapp", "[Unit]\nDescription="+tc.previous+"\n")

	if err := tc.write(context.Background(), "myapp", appDir); err != nil {
		t.Fatalf("writing the unit over an outgoing unit failed: %v", err)
	}
	return recordedCalls(t, logPath)
}

// stubSystemctl puts a recording `systemctl` first on PATH and returns the file it logs to, so the
// handover's order can be asserted without a real user manager. It reads the named application's
// unit file, resolved here rather than inside the stub, so the caller's XDG configuration is the
// one that decides which file a stop is recorded against. Alongside each stop it records the
// description that file carried at that moment, which is what tells a stop that ran before the unit
// was replaced from one that ran after it.
func stubSystemctl(t *testing.T, appName string) string {
	t.Helper()
	dir := t.TempDir()
	logPath := filepath.Join(dir, "systemctl.log")
	script := "#!/bin/sh\nprintf '%s\\n' \"$*\" >> \"$GARE_TEST_SYSTEMCTL_LOG\"\n" +
		"[ \"$1\" = --user ] && [ \"$2\" = stop ] && sed -n 's/^Description=//p'" +
		" \"$GARE_TEST_UNIT_PATH\" >> \"$GARE_TEST_SYSTEMCTL_LOG\"\nexit 0\n"
	if err := os.WriteFile(filepath.Join(dir, "systemctl"), []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("GARE_TEST_SYSTEMCTL_LOG", logPath)
	t.Setenv("GARE_TEST_UNIT_PATH", systemd.GetUnitPath(appName))
	return logPath
}
