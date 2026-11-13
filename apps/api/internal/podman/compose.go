package podman

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

// ComposeDownOptions contains options for tearing down a compose stack.
type ComposeDownOptions struct {
	RepoDir     string
	ComposeFile string
	Project     string
	Volumes     bool
	Stdout      io.Writer
	Stderr      io.Writer
}

// CheckComposeProvider verifies that podman can reach an external compose provider.
func CheckComposeProvider(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "podman", "compose", "version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("podman compose provider unavailable (install docker-compose or podman-compose): %w: %s",
			err, strings.TrimSpace(string(output)))
	}
	return nil
}

// ComposeProjectContainers returns the IDs of containers in a compose project.
func ComposeProjectContainers(ctx context.Context, project string) ([]string, error) {
	args := []string{"ps", "-a", "--filter", "label=com.docker.compose.project=" + project, "--format", "{{.ID}}"}
	cmd := exec.CommandContext(ctx, "podman", args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	return strings.Fields(string(output)), nil
}

// ComposeDown stops a compose stack, optionally removing its named volumes.
func ComposeDown(ctx context.Context, opts ComposeDownOptions) error {
	args := []string{"compose", "-f", opts.ComposeFile, "-p", opts.Project, "down"}
	if opts.Volumes {
		args = append(args, "-v")
	}
	cmd := exec.CommandContext(ctx, "podman", args...)
	cmd.Dir = opts.RepoDir
	cmd.Stdout = opts.Stdout
	cmd.Stderr = opts.Stderr
	return cmd.Run()
}
