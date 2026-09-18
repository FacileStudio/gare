package caddy

import (
	"errors"
	"os"
	"strings"
	"syscall"
)

// ResolveConfDir returns the active Caddy snippet directory, allowing GARE_CADDY_CONF_DIR override.
func ResolveConfDir() string {
	if dir := os.Getenv("GARE_CADDY_CONF_DIR"); dir != "" {
		return dir
	}
	return DefaultConfDir
}

// ResolveCaddyfilePath returns the active Caddyfile path, allowing GARE_CADDYFILE override.
func ResolveCaddyfilePath() string {
	if path := os.Getenv("GARE_CADDYFILE"); path != "" {
		return path
	}
	return DefaultCaddyfile
}

// CheckReloadError filters out non-fatal errors such as connection refused when Caddy daemon is offline.
func CheckReloadError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, syscall.ECONNREFUSED) || strings.Contains(err.Error(), "connection refused") {
		return nil
	}
	return err
}
