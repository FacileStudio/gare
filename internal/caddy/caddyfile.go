package caddy

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/FacileStudio/gare/internal/atomicfile"
)

// EnsureCaddyfile creates or updates the default root Caddyfile to include snippets.
func EnsureCaddyfile() error {
	return EnsureCaddyfilePaths(ResolveCaddyfilePath(), ResolveConfDir())
}

// EnsureCaddyfilePaths creates or updates a root Caddyfile ensuring the snippet directory is imported.
func EnsureCaddyfilePaths(caddyfilePath, confDir string) error {
	if confDir == "" {
		confDir = ResolveConfDir()
	}
	if caddyfilePath == "" {
		caddyfilePath = ResolveCaddyfilePath()
	}
	tryMkdir(confDir)
	tryMkdir(filepath.Dir(caddyfilePath))

	content, err := os.ReadFile(caddyfilePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			defaultContent := fmt.Sprintf("import %s/*.caddy\n", confDir)
			return writeCaddyfile(caddyfilePath, []byte(defaultContent))
		}
		return err
	}

	if hasConfDirImport(string(content), confDir) {
		return nil
	}

	updated := appendImportDirective(string(content), confDir)
	return writeCaddyfile(caddyfilePath, []byte(updated))
}

func hasConfDirImport(content, confDir string) bool {
	for line := range strings.SplitSeq(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "import ") {
			if strings.Contains(trimmed, confDir) || strings.Contains(trimmed, "conf.d") {
				return true
			}
		}
	}
	return false
}

func appendImportDirective(content, confDir string) string {
	trimmed := strings.TrimSpace(content)
	directive := fmt.Sprintf("import %s/*.caddy", confDir)
	if trimmed == "" {
		return directive + "\n"
	}
	return trimmed + "\n\n" + directive + "\n"
}

func writeCaddyfile(path string, data []byte) error {
	if err := atomicfile.WriteFile(path, data, 0644); err != nil {
		return handleWriteError(path, err)
	}
	return nil
}

func handleWriteError(path string, err error) error {
	if errors.Is(err, syscall.ENOSPC) {
		return fmt.Errorf("could not write %s: disk full", path)
	}
	if errors.Is(err, syscall.EDQUOT) {
		return fmt.Errorf("could not write %s: quota exceeded", path)
	}
	return fmt.Errorf("could not write %s: %w", path, err)
}

func tryMkdir(dir string) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return
	}
}
