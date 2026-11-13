// Package git runs the git commands gare needs to fetch and inspect an application repository.
package git

import (
	"context"
	"fmt"
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

// Clone clones a git repository using the provided options. The URL and target directory follow
// `--`, so a URL that starts with a dash reaches git as a repository and never as an option.
func Clone(ctx context.Context, opts CloneOptions) error {
	args := append(opts.Auth.args(), "clone")
	if opts.Branch != "" {
		if err := ValidateBranch(ctx, opts.Branch); err != nil {
			return err
		}
		args = append(args, "--branch", opts.Branch)
	}
	args = append(args, "--", opts.RepoURL, opts.TargetDir)
	return run(ctx, args, opts.Stdout, opts.Stderr)
}

// SyncOptions describes the checkout to align with its remote and where to stream git's output.
type SyncOptions struct {
	RepoDir string
	Branch  string
	Auth    Auth
	Stdout  io.Writer
	Stderr  io.Writer
}

// Sync makes the checkout at RepoDir match origin's Branch exactly. It fetches with a forced
// refspec, force-checks out the fetched commit and removes untracked files, so a force-pushed
// remote or a build that modified a tracked file cannot leave the checkout stuck the way a
// `git pull` would. Ignored files stay, so cached build output survives.
func Sync(ctx context.Context, opts SyncOptions) error {
	branch, err := syncBranch(ctx, opts)
	if err != nil {
		return err
	}
	remoteRef := "refs/remotes/origin/" + branch
	fetch := append([]string{"-C", opts.RepoDir}, opts.Auth.args()...)
	fetch = append(fetch, "fetch", "--prune", "origin", "+refs/heads/"+branch+":"+remoteRef)
	steps := [][]string{
		fetch,
		{"-C", opts.RepoDir, "checkout", "--force", "-B", branch, remoteRef},
		{"-C", opts.RepoDir, "clean", "-fd"},
	}
	for _, args := range steps {
		if err := run(ctx, args, opts.Stdout, opts.Stderr); err != nil {
			return err
		}
	}
	return nil
}

// syncBranch returns the branch Sync aligns, falling back to the checked-out branch for app
// configs written before gare recorded one, and validates it before any git command uses it.
func syncBranch(ctx context.Context, opts SyncOptions) (string, error) {
	branch := opts.Branch
	if branch == "" {
		out, err := exec.CommandContext(ctx, "git", "-C", opts.RepoDir, "symbolic-ref", "--short", "HEAD").Output()
		if err != nil {
			return "", fmt.Errorf("resolve the checked-out branch of %s: %w", opts.RepoDir, err)
		}
		branch = strings.TrimSpace(string(out))
	}
	if err := ValidateBranch(ctx, branch); err != nil {
		return "", err
	}
	return branch, nil
}

// ValidateBranch rejects a branch name git would read as an option or refuse as a ref, so a
// hostile or mistyped name fails before any git command runs with it.
func ValidateBranch(ctx context.Context, branch string) error {
	if branch == "" || strings.HasPrefix(branch, "-") {
		return fmt.Errorf("invalid git branch %q", branch)
	}
	if err := exec.CommandContext(ctx, "git", "check-ref-format", "--branch", branch).Run(); err != nil {
		return fmt.Errorf("invalid git branch %q: %w", branch, err)
	}
	return nil
}

// run executes git with gare's non-interactive environment and streams its output, because a
// clone or fetch can take long enough that the operator needs to watch it.
func run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
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
