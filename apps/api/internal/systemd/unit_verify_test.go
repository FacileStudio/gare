package systemd

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// verifyUnitWithSystemd asks systemd itself whether a generated unit parses. This is the check
// that replaced the Podman generator's dry run: gare now spells the unit out, so nothing else
// validates it before systemd does.
func verifyUnitWithSystemd(t *testing.T, unitPath string) {
	t.Helper()
	if _, err := exec.LookPath("systemd-analyze"); err != nil {
		skipUnlessPodmanUnits(t, "systemd-analyze is not installed: %v", err)
	}
	if _, err := exec.LookPath("podman"); err != nil {
		skipUnlessPodmanUnits(t, "podman is not installed, so the unit's ExecStart cannot resolve: %v", err)
	}

	cmd := exec.Command("systemd-analyze", "--user", "verify", unitPath)
	cmd.Env = analyzeEnv(t)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("systemd-analyze could not verify the unit: %v\n%s", err, output)
	}
	if complaints := unitComplaints(string(output), unitPath); complaints != "" {
		t.Fatalf("systemd rejected the generated unit:\n%s", complaints)
	}
}

// unitComplaints returns the verify output lines systemd attributed to the unit itself.
// systemd-analyze still exits zero when it rejects a directive, so the messages are the only
// signal that a unit is malformed, and they are what the caller has to assert on.
func unitComplaints(output, unitPath string) string {
	prefix := unitPath + ":"
	var complaints []string
	for line := range strings.SplitSeq(output, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), prefix) {
			complaints = append(complaints, line)
		}
	}
	return strings.Join(complaints, "\n")
}

// analyzeEnv guarantees XDG_RUNTIME_DIR. Without it systemd-analyze --user fails to initialize
// its manager and reports an error that has nothing to do with the unit under test.
func analyzeEnv(t *testing.T) []string {
	t.Helper()
	if os.Getenv("XDG_RUNTIME_DIR") != "" {
		return os.Environ()
	}
	return append(os.Environ(), "XDG_RUNTIME_DIR="+t.TempDir())
}

// skipUnlessPodmanUnits lets a machine without the podman runtime skip these tests, while CI
// fails loudly: the point of them is to prove the generated units are accepted, so a silent skip
// there would hide exactly the regression they exist to catch.
func skipUnlessPodmanUnits(t *testing.T, format string, args ...any) {
	t.Helper()
	if os.Getenv("GARE_REQUIRE_PODMAN") == "1" {
		t.Fatalf(format, args...)
	}
	t.Skipf(format, args...)
}

func TestUnitComplaintsReportsOnlyTheVerifiedUnit(t *testing.T) {
	unitPath := "/home/user/.config/systemd/user/myapp.service"
	output := strings.Join([]string{
		"other.service: Failed to open /etc/systemd/system/other.service: Permission denied",
		unitPath + ":11: Failed to parse Restart=on-failureX, ignoring: Invalid argument",
		"",
	}, "\n")

	complaints := unitComplaints(output, unitPath)
	if !strings.Contains(complaints, "Failed to parse Restart=on-failureX") {
		t.Errorf("expected the unit's own complaint to be reported, got %q", complaints)
	}
	if strings.Contains(complaints, "other.service") {
		t.Errorf("unrelated units must not fail this unit's check, got %q", complaints)
	}
}

func TestUnitComplaintsIsEmptyForACleanUnit(t *testing.T) {
	if complaints := unitComplaints("", "/home/user/.config/systemd/user/myapp.service"); complaints != "" {
		t.Errorf("expected no complaints, got %q", complaints)
	}
}
