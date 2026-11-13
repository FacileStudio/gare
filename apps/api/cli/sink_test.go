package cli

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/FacileStudio/gare/apps/api/internal/engine"
)

// sinkHelperEnv makes the test binary act as a process that only renders engine events, so the
// parent test can read what reached the real stdout and stderr: the styles write to the process's
// own streams, bound when the process starts.
const sinkHelperEnv = "GARE_TEST_TERMINAL_SINK"

// TestTerminalSinkKeepsTheOutputOfEarlierReleases pins what moving the lifecycle into the engine
// must not change: each kind of progress line keeps the icon and the stream gare printed it with
// before, steps, finished steps and details on stdout and warnings on stderr.
func TestTerminalSinkKeepsTheOutputOfEarlierReleases(t *testing.T) {
	if os.Getenv(sinkHelperEnv) == "1" {
		renderEveryKind()
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestTerminalSinkKeepsTheOutputOfEarlierReleases$")
	cmd.Env = append(os.Environ(), sinkHelperEnv+"=1", "NO_COLOR=1")
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("the sink helper failed: %v\n%s", err, stderr.String())
	}
	if want := "▸ Syncing app\n✓ Synced manifest.yaml from repository\n· Wrote systemd unit\nbuild output\n"; !strings.Contains(stdout.String(), want) {
		t.Errorf("stdout = %q, want it to contain %q", stdout.String(), want)
	}
	if want := "! port busy\nbuild error\n"; stderr.String() != want {
		t.Errorf("stderr = %q, want %q", stderr.String(), want)
	}
}

// renderEveryKind sends one event of each kind, then a line of subprocess output on each stream,
// through the terminal sink.
func renderEveryKind() {
	sink := terminalSink{}
	sink.Event("app", engine.KindStep, "Syncing app")
	sink.Event("app", engine.KindDone, "Synced manifest.yaml from repository")
	sink.Event("app", engine.KindWarn, "port busy")
	sink.Event("app", engine.KindDetail, "Wrote systemd unit")
	stdout, stderr := sink.Output("app")
	stdout.Write([]byte("build output\n"))
	stderr.Write([]byte("build error\n"))
}
