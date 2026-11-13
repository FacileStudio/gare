package podman

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// Tag gives the local image src the additional name dst.
func Tag(ctx context.Context, src, dst string) error {
	output, err := exec.CommandContext(ctx, "podman", "tag", src, dst).CombinedOutput()
	if err != nil {
		return fmt.Errorf("podman tag %s %s: %w: %s", src, dst, err, strings.TrimSpace(string(output)))
	}
	return nil
}

// ImageTags returns every tag local storage holds for the image repository repo.
func ImageTags(ctx context.Context, repo string) ([]string, error) {
	cmd := exec.CommandContext(ctx, "podman", "images", "--noheading", "--format", "{{.Tag}}", repo)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list the tags of %s: %w", repo, err)
	}
	return strings.Fields(string(output)), nil
}
