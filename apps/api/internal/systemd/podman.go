package systemd

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// CheckPodmanWorkloads verifies the installed podman accepts the service-container flags gare's
// units hand to "podman kube play". Podman older than 5.0 rejects them, and discovering that here
// fails the deploy before a running workload is stopped rather than at unit start.
func CheckPodmanWorkloads(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, ResolvePodmanPath(), "kube", "play",
		"--service-container=true", "--service-exit-code-propagation=any", "--help")
	cmd.Env = userEnviron()
	output, err := cmd.CombinedOutput()
	if err == nil {
		return nil
	}
	return fmt.Errorf("podman kube play rejected the service-container flags gare's units pass it "+
		"(%s): podman 5.0 or newer is required", firstLine(output, err))
}

// firstLine reduces a command failure to one line of detail, preferring podman's own diagnostic
// over the bare exit status so the message says what podman objected to.
func firstLine(output []byte, err error) string {
	detail, _, _ := strings.Cut(strings.TrimSpace(string(output)), "\n")
	if detail == "" {
		return err.Error()
	}
	return detail
}
