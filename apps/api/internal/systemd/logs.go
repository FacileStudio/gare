package systemd

import (
	"context"
	"errors"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

// DefaultLogLines is how many journal lines a command shows when the caller names no count.
const DefaultLogLines = 100

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
		lines = DefaultLogLines
	}
	if lines > 0 {
		args = append(args, "-n", strconv.Itoa(lines))
	}
	return args
}

// RecentLogs returns the most recent journal lines for a unit, so a failed start can report what
// actually went wrong instead of systemd's generic "control process exited with error code".
// Diagnostics are best-effort: an unreadable journal yields no lines rather than another error.
func RecentLogs(ctx context.Context, name string, lines int) string {
	cmd := exec.CommandContext(ctx, "journalctl", buildLogArgs(name, LogOptions{Lines: lines})...)
	cmd.Env = userEnviron()
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
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
