package main

import (
	"context"
	"fmt"

	"github.com/FacileStudio/gare/internal/podman"
	"github.com/FacileStudio/gare/internal/systemd"
)

func checkComposeProvider(ctx context.Context) {
	if err := podman.CheckComposeProvider(ctx); err != nil {
		printWarning("podman compose provider is not available")
		fmt.Printf("\nCompose workloads need an external provider. Install the podman-compose package with:\n" +
			"  sudo apt install podman-compose\n")
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
	installed, err := systemd.UnitFileInstalled(ctx, "podman.socket")
	return err == nil && installed
}
