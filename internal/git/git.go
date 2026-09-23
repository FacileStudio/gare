// Package git runs the git commands gare needs to fetch and inspect an application repository.
package git

import (
	"context"
	"io"
	"os/exec"
	"strings"
)

// CloneOptions contains options for cloning a Git repository.
type CloneOptions struct {
	RepoURL   string
	Branch    string
	TargetDir string
	Auth      Auth
	Stdout    io.Writer
	Stderr    io.Writer
}

// Clone clones a git repository using the provided options.
func Clone(ctx context.Context, opts CloneOptions) error {
	args := opts.Auth.args()
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
func Pull(ctx context.Context, repoDir string, auth Auth, stdout, stderr io.Writer) error {
	args := []string{"-C", repoDir}
	args = append(args, auth.args()...)
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
