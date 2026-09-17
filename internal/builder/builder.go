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

// BuildOptions defines configuration for building a container image.
type BuildOptions struct {
	RepoDir       string
	ImageName     string
	Containerfile string
	ContextDir    string
	Stdout        io.Writer
	Stderr        io.Writer
}

// Clone clones a git repository using the provided options.
func Clone(ctx context.Context, opts CloneOptions) error {
	args := gitAuthArgs()
	if opts.Branch != "" {
		args = append(args, "clone", "--branch", opts.Branch, opts.RepoURL, opts.TargetDir)
	} else {
		args = append(args, "clone", opts.RepoURL, opts.TargetDir)
	}
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Env = gitEnviron()
	cmd.Stdout = opts.Stdout
	cmd.Stderr = opts.Stderr
	return cmd.Run()
}

// Pull updates the git repository at repoDir by running git pull.
func Pull(ctx context.Context, repoDir string, stdout, stderr io.Writer) error {
	args := []string{"-C", repoDir}
	args = append(args, gitAuthArgs()...)
	args = append(args, "pull")
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Env = gitEnviron()
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
