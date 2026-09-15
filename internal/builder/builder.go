package builder

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// CloneOptions contains options for cloning a Git repository.
type CloneOptions struct {
	RepoURL   string
	Branch    string
	TargetDir string
	Stdout    io.Writer
	Stderr    io.Writer
}

// Clone clones a git repository using the provided options.
func Clone(ctx context.Context, opts CloneOptions) error {
	var args []string
	if opts.Branch != "" {
		args = []string{"clone", "--branch", opts.Branch, opts.RepoURL, opts.TargetDir}
	} else {
		args = []string{"clone", opts.RepoURL, opts.TargetDir}
	}
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Stdout = opts.Stdout
	cmd.Stderr = opts.Stderr
	return cmd.Run()
}

// Pull updates the git repository at repoDir by running git pull.
func Pull(ctx context.Context, repoDir string, stdout, stderr io.Writer) error {
	cmd := exec.CommandContext(ctx, "git", "-C", repoDir, "pull")
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}

// GetCommitHash retrieves the abbreviated HEAD commit hash for the repository at repoDir.
func GetCommitHash(ctx context.Context, repoDir string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", repoDir, "rev-parse", "--short", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// DetectContainerfile checks for Containerfile or Dockerfile in repoDir and returns its filename.
func DetectContainerfile(repoDir string) (string, error) {
	for _, name := range []string{"Containerfile", "Dockerfile"} {
		path := filepath.Join(repoDir, name)
		info, err := os.Stat(path)
		if err == nil && !info.IsDir() {
			return name, nil
		}
	}
	return "", fmt.Errorf("neither Containerfile nor Dockerfile found in %s", repoDir)
}

// Build compiles a container image with the specified tag using podman build.
func Build(ctx context.Context, repoDir, imageName string, stdout, stderr io.Writer) error {
	cmd := exec.CommandContext(ctx, "podman", "build", "-t", imageName, ".")
	cmd.Dir = repoDir
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}

// PruneImages cleans up dangling container images via podman image prune.
func PruneImages(ctx context.Context, stdout, stderr io.Writer) error {
	cmd := exec.CommandContext(ctx, "podman", "image", "prune", "-f")
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}

// RemoveImage forcefully deletes the specified container image by name.
func RemoveImage(ctx context.Context, imageName string) error {
	cmd := exec.CommandContext(ctx, "podman", "rmi", "-f", imageName)
	return cmd.Run()
}
