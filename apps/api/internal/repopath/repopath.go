// Package repopath keeps a path that a repository or its operator configures inside that repository.
// gare mounts, builds and runs what these paths name, so a path that escapes the checkout would let
// one app serve or build another app's directory, secrets included.
package repopath

import (
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"path/filepath"
	"slices"
	"strings"
)

// Validate rejects a path that cannot stay inside the repository it is relative to: an empty one,
// one with whitespace, since systemd splits unit lines on it, an absolute one, and one that climbs
// out through a ".." element once cleaned. key names the setting in the error so the operator knows
// which line to fix.
func Validate(key, path string) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("invalid %s: a path is required", key)
	}
	if strings.ContainsAny(path, " \t\r\n") {
		return fmt.Errorf("invalid %s %q: whitespace is not allowed", key, path)
	}
	cleaned := filepath.Clean(path)
	if filepath.IsAbs(cleaned) || slices.Contains(strings.Split(cleaned, string(filepath.Separator)), "..") {
		return fmt.Errorf("invalid %s %q: must be a relative path inside the repository", key, path)
	}
	return nil
}

// ValidateSet validates every set path in fields, keyed by the setting that holds it, in key order
// so the error is stable. An empty value means the setting is unset and is skipped, which is what
// lets gare.yml keys and create flags stay optional while sharing the rule.
func ValidateSet(fields map[string]string) error {
	for _, key := range slices.Sorted(maps.Keys(fields)) {
		if fields[key] == "" {
			continue
		}
		if err := Validate(key, fields[key]); err != nil {
			return err
		}
	}
	return nil
}

// Join validates path, joins it to repoDir and checks that what it names on disk, once every
// symlink is followed, is still inside repoDir, because a lexically clean path can be a symlink the
// repository ships to anywhere on the host. A path that does not exist yet passes: it escapes
// nothing, and the next deploy checks it again once a build has created it. An empty path names
// the repository root, as it did for apps an older gare recorded without one.
func Join(key, repoDir, path string) (string, error) {
	if path == "" {
		path = "."
	}
	if err := Validate(key, path); err != nil {
		return "", err
	}
	joined := filepath.Join(repoDir, path)
	root, err := filepath.EvalSymlinks(repoDir)
	if err != nil {
		return "", fmt.Errorf("failed to resolve the repository directory: %w", err)
	}
	target, err := filepath.EvalSymlinks(joined)
	if errors.Is(err, fs.ErrNotExist) {
		return joined, nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to resolve %s %q: %w", key, path, err)
	}
	rel, err := filepath.Rel(root, target)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("invalid %s %q: resolves to %s, outside the repository", key, path, target)
	}
	return joined, nil
}
