package systemd

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// systemctlCmd builds a systemctl invocation in the current user's session, which is where every
// unit gare writes lives.
func systemctlCmd(ctx context.Context, args ...string) *exec.Cmd {
	fullArgs := append([]string{"--user"}, args...)
	cmd := exec.CommandContext(ctx, "systemctl", fullArgs...)
	cmd.Env = userEnviron()
	return cmd
}

// runSystemctl invokes a systemctl user verb, folding systemctl's own diagnostic into the error so
// a failed command reports why it failed rather than a bare exit status.
func runSystemctl(ctx context.Context, args ...string) error {
	cmd := systemctlCmd(ctx, args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

// DaemonReload triggers a systemd user daemon reload.
func DaemonReload(ctx context.Context) error {
	return runSystemctl(ctx, "daemon-reload")
}

// Enable enables the specified user service for auto-start.
func Enable(ctx context.Context, name string) error {
	return runSystemctl(ctx, "enable", name+".service")
}

// Restart restarts the specified user service.
func Restart(ctx context.Context, name string) error {
	return runSystemctl(ctx, "restart", name+".service")
}

// Stop stops the specified user service.
func Stop(ctx context.Context, name string) error {
	return runSystemctl(ctx, "stop", name+".service")
}

// Disable disables the specified user service from auto-starting.
func Disable(ctx context.Context, name string) error {
	return runSystemctl(ctx, "disable", name+".service")
}
