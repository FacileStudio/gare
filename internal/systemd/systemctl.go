package systemd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// DaemonReload triggers a systemd user daemon reload.
func DaemonReload(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "systemctl", "--user", "daemon-reload")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

// EnableAndStart enables and immediately starts the specified user service.
func EnableAndStart(ctx context.Context, name string) error {
	cmd := exec.CommandContext(ctx, "systemctl", "--user", "enable", "--now", name+".service")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

// Enable enables the specified user service for auto-start.
func Enable(ctx context.Context, name string) error {
	cmd := exec.CommandContext(ctx, "systemctl", "--user", "enable", name+".service")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

// Restart restarts the specified user service.
func Restart(ctx context.Context, name string) error {
	cmd := exec.CommandContext(ctx, "systemctl", "--user", "restart", name+".service")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

// Stop stops the specified user service.
func Stop(ctx context.Context, name string) error {
	cmd := exec.CommandContext(ctx, "systemctl", "--user", "stop", name+".service")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

// Disable disables the specified user service from auto-starting.
func Disable(ctx context.Context, name string) error {
	cmd := exec.CommandContext(ctx, "systemctl", "--user", "disable", name+".service")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

// IsActive checks if the specified systemd user service is currently active.
func IsActive(ctx context.Context, name string) (string, error) {
	cmd := exec.CommandContext(ctx, "systemctl", "--user", "is-active", name+".service")
	output, err := cmd.CombinedOutput()
	status := strings.TrimSpace(string(output))
	if err != nil && status == "" {
		return "inactive", err
	}
	return status, nil
}

// CheckLinger checks whether logind linger is enabled for the specified user.
func CheckLinger(ctx context.Context, user string) (bool, error) {
	if user == "" {
		user = os.Getenv("USER")
	}
	cmd := exec.CommandContext(ctx, "loginctl", "show-user", user, "--property=Linger")
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}
	val := strings.TrimSpace(string(output))
	return strings.EqualFold(val, "Linger=yes"), nil
}
