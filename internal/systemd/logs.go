package systemd

import (
	"context"
	"errors"
	"io"
	"os/exec"
	"strconv"
	"syscall"
)

const defaultLogLines = 100

// LogOptions defines options for streaming systemd journal logs.
type LogOptions struct {
	Follow bool
	Lines  int
	Stdout io.Writer
	Stderr io.Writer
}

// StreamLogs streams systemd journal logs for the specified service to the given writers.
func StreamLogs(ctx context.Context, name string, opts LogOptions) error {
	args := buildLogArgs(name, opts)
	cmd := exec.CommandContext(ctx, "journalctl", args...)
	cmd.Env = userEnviron()
	cmd.Stdout = opts.Stdout
	cmd.Stderr = opts.Stderr
	if err := cmd.Run(); err != nil && !isInterrupt(ctx, err) {
		return err
	}
	return nil
}

func buildLogArgs(name string, opts LogOptions) []string {
	args := []string{"--no-pager", "--user", "-u", name + ".service"}
	if opts.Follow {
		args = append(args, "-f")
	}
	lines := opts.Lines
	if lines == 0 {
		lines = defaultLogLines
	}
	if lines > 0 {
		args = append(args, "-n", strconv.Itoa(lines))
	}
	return args
}

func isInterrupt(ctx context.Context, err error) bool {
	if errors.Is(err, context.Canceled) || errors.Is(ctx.Err(), context.Canceled) {
		return true
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		return false
	}
	return isSignalExit(exitErr)
}

func isSignalExit(exitErr *exec.ExitError) bool {
	if exitErr.ExitCode() == 130 {
		return true
	}
	status, ok := exitErr.Sys().(syscall.WaitStatus)
	if !ok || !status.Signaled() {
		return false
	}
	sig := status.Signal()
	return sig == syscall.SIGINT || sig == syscall.SIGTERM
}


