package systemd

import (
	"context"
	"io"
	"os/exec"
	"strconv"
)

// LogOptions defines options for streaming systemd journal logs.
type LogOptions struct {
	Follow bool
	Lines  int
	Stdout io.Writer
	Stderr io.Writer
}

// StreamLogs streams systemd journal logs for the specified service to the given writers.
func StreamLogs(ctx context.Context, name string, opts LogOptions) error {
	args := []string{"--no-pager", "--user", "-u", name + ".service"}
	if opts.Follow {
		args = append(args, "-f")
	}
	if opts.Lines > 0 {
		args = append(args, "-n", strconv.Itoa(opts.Lines))
	}
	cmd := exec.CommandContext(ctx, "journalctl", args...)
	cmd.Env = userEnviron()
	cmd.Stdout = opts.Stdout
	cmd.Stderr = opts.Stderr
	return cmd.Run()
}
