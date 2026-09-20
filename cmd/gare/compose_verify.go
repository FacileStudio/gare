package main

import (
	"context"
	"fmt"

	"github.com/FacileStudio/gare/internal/builder"
	"github.com/FacileStudio/gare/internal/storage"
)

func verifyComposeContainersGone(ctx context.Context, cfg *storage.AppConfig, action string) {
	if !cfg.IsCompose() {
		return
	}
	leftovers, err := builder.ComposeProjectContainers(ctx, composeProjectName(cfg.Name))
	if err != nil || len(leftovers) == 0 {
		return
	}
	printWarning(fmt.Sprintf("%d compose container(s) still present after %s for %s; inspect with: podman ps -a",
		len(leftovers), action, cfg.Name))
}
