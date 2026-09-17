package systemd

import (
	"context"
	"os"
	"os/exec"
	"os/user"
	"strconv"
)

func currentUID() string {
	if u, err := user.Current(); err == nil && u.Uid != "" {
		return u.Uid
	}
	return strconv.Itoa(os.Getuid())
}

func currentUsername() string {
	if u, err := user.Current(); err == nil && u.Username != "" {
		return u.Username
	}
	if val := os.Getenv("USER"); val != "" {
		return val
	}
	if val := os.Getenv("LOGNAME"); val != "" {
		return val
	}
	return currentUID()
}

func userEnviron() []string {
	env := os.Environ()
	uid := currentUID()
	runtimeDir := os.Getenv("XDG_RUNTIME_DIR")
	if runtimeDir == "" {
		runtimeDir = "/run/user/" + uid
		env = append(env, "XDG_RUNTIME_DIR="+runtimeDir)
	}
	if os.Getenv("DBUS_SESSION_BUS_ADDRESS") == "" {
		env = append(env, "DBUS_SESSION_BUS_ADDRESS=unix:path="+runtimeDir+"/bus")
	}
	return env
}

func systemctlCmd(ctx context.Context, args ...string) *exec.Cmd {
	fullArgs := append([]string{"--user"}, args...)
	cmd := exec.CommandContext(ctx, "systemctl", fullArgs...)
	cmd.Env = userEnviron()
	return cmd
}
