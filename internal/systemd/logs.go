package systemd

import (
	"context"
	"io"
	"os/exec"
)

// StreamLogs streams systemd journal logs for the specified service to the given writers.
func StreamLogs(ctx context.Context, name string, follow bool, stdout, stderr io.Writer) error {
	args := []string{"--no-pager", "--user", "-u", name + ".service"}
	if follow {
		args = append(args, "-f")
	}
	cmd := exec.CommandContext(ctx, "journalctl", args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}
