package systemd

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// IsActive checks if the specified systemd user service is currently active.
func IsActive(ctx context.Context, name string) (string, error) {
	cmd := systemctlCmd(ctx, "is-active", name+".service")
	output, err := cmd.CombinedOutput()
	status := strings.TrimSpace(string(output))
	if err != nil && status == "" {
		return "inactive", err
	}
	return status, nil
}

// UnitFileInstalled reports whether systemd knows the named unit file, matching it whether it is
// enabled, disabled or only present, because a workload can need a unit installed rather than
// started. It runs in the current user's session like every other query here, so a caller outside a
// login shell still reaches the user manager instead of failing to connect.
func UnitFileInstalled(ctx context.Context, name string) (bool, error) {
	cmd := systemctlCmd(ctx, "list-unit-files", name, "--no-legend")
	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("failed to list systemd unit files: %w", err)
	}
	return strings.TrimSpace(string(output)) != "", nil
}

// CheckLinger checks whether logind linger is enabled for the specified user.
func CheckLinger(ctx context.Context, user string) (bool, error) {
	if user == "" {
		user = currentUsername()
	}
	cmd := exec.CommandContext(ctx, "loginctl", "show-user", user, "--property=Linger")
	cmd.Env = userEnviron()
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}
	val := strings.TrimSpace(string(output))
	return strings.EqualFold(val, "Linger=yes"), nil
}
