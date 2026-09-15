package caddy

import (
	"errors"
	"fmt"
	"os"
	"syscall"

	"github.com/FacileStudio/gare/internal/atomicfile"
)

// EnsureCaddyfile creates a default root Caddyfile if one is missing.
func EnsureCaddyfile() error {
	return EnsureCaddyfileAtPath(DefaultCaddyfile)
}

// EnsureCaddyfileAtPath creates a default root Caddyfile at the specified path if missing.
func EnsureCaddyfileAtPath(path string) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	if err := atomicfile.WriteFile(path, []byte(defaultCaddyfileContent), 0644); err != nil {
		if errors.Is(err, syscall.ENOSPC) {
			return fmt.Errorf("could not write %s: disk full", path)
		}
		if errors.Is(err, syscall.EDQUOT) {
			return fmt.Errorf("could not write %s: quota exceeded", path)
		}
		return fmt.Errorf("could not write %s: %w", path, err)
	}
	return nil
}
