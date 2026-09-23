// Package podman runs the podman commands gare shells out to, other than unit execution.
package podman

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// BuildOptions defines configuration for building a container image.
type BuildOptions struct {
	RepoDir       string
	ImageName     string
	Containerfile string
	ContextDir    string
	Stdout        io.Writer
	Stderr        io.Writer
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

// Build compiles a container image with the specified options using podman build.
func Build(ctx context.Context, opts BuildOptions) error {
	cf := opts.Containerfile
	if cf == "" {
		if detected, err := DetectContainerfile(opts.RepoDir); err == nil {
			cf = detected
		}
	}
	contextDir := opts.ContextDir
	if contextDir == "" {
		contextDir = "."
	}
	args := []string{"build", "-t", opts.ImageName}
	if cf != "" {
		args = append(args, "-f", cf)
	}
	args = append(args, contextDir)
	cmd := exec.CommandContext(ctx, "podman", args...)
	cmd.Dir = opts.RepoDir
	cmd.Stdout = opts.Stdout
	cmd.Stderr = opts.Stderr
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

// RunBuildCommand runs an arbitrary build command string using sh -c within dir.
func RunBuildCommand(ctx context.Context, dir, command string, stdout, stderr io.Writer) error {
	trimmed := strings.TrimSpace(command)
	if trimmed == "" {
		return nil
	}
	cmd := exec.CommandContext(ctx, "sh", "-c", trimmed)
	cmd.Dir = dir
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}
