package main

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/FacileStudio/gare/internal/podman"
)

func checkComposeProvider(ctx context.Context) {
	if err := podman.CheckComposeProvider(ctx); err != nil {
		printWarning("podman compose provider is not available")
		fmt.Printf("\nCompose workloads need an external provider. Install one with:\n" +
			"  sudo apt install docker-compose-v2\n  # or\n  pipx install podman-compose\n\n")
	} else {
		printSuccess("Found podman compose provider")
	}

	if !podmanSocketAvailable(ctx) {
		printWarning("podman.socket is not installed; compose workloads require it")
		return
	}
	printSuccess("Found podman.socket")
}

func podmanSocketAvailable(ctx context.Context) bool {
	cmd := exec.CommandContext(ctx, "systemctl", "--user", "list-unit-files", "podman.socket", "--no-legend")
	output, err := cmd.Output()
	return err == nil && strings.TrimSpace(string(output)) != ""
}
