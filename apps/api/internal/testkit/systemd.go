package testkit

import (
	"os"
	"path/filepath"
	"testing"
)

// systemctlStub answers like a user manager holding one unit whose state lives in a file: start
// and restart make it active, stop inactive, and show and is-active report it. Every call is
// logged, and any other verb succeeds without effect.
const systemctlStub = `#!/bin/sh
printf '%s\n' "$*" >> "$GARE_TEST_SYSTEMCTL_LOG"
verb=""
for a in "$@"; do case "$a" in -*) ;; *) verb="$a"; break ;; esac; done
case "$verb" in
start|restart) echo active > "$GARE_TEST_SYSTEMD_STATE" ;;
stop) echo inactive > "$GARE_TEST_SYSTEMD_STATE" ;;
show) printf 'ActiveState=%s\nSubState=running\nResult=success\nMainPID=0\n' "$(cat "$GARE_TEST_SYSTEMD_STATE")" ;;
is-active) cat "$GARE_TEST_SYSTEMD_STATE" ;;
esac
exit 0
`

// journalctlStub prints two journal lines and exits, as a follow would once the client left.
const journalctlStub = "#!/bin/sh\nprintf 'first line\\nsecond line\\n'\n"

// StubCommand puts an executable script named name first on PATH for the rest of the test.
func StubCommand(t *testing.T, name, script string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// StubSystemd replaces systemctl and journalctl with the stubs above, the unit starting inactive,
// and points the user's config home at a temp directory so no unit file reaches the real one. It
// returns the file systemctl logs its calls to.
func StubSystemd(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	state := filepath.Join(dir, "state")
	if err := os.WriteFile(state, []byte("inactive\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GARE_TEST_SYSTEMD_STATE", state)
	t.Setenv("GARE_TEST_SYSTEMCTL_LOG", filepath.Join(dir, "systemctl.log"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, "config"))
	StubCommand(t, "systemctl", systemctlStub)
	StubCommand(t, "journalctl", journalctlStub)
	return filepath.Join(dir, "systemctl.log")
}
