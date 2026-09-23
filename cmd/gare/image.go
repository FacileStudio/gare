package main

import (
	"context"
	"fmt"

	"github.com/FacileStudio/gare/internal/podman"
	"github.com/FacileStudio/gare/internal/storage"
)

// appImageRef is the tag gare builds and runs an application image under. The localhost/ prefix
// marks it as locally built, which is what tells podman never to resolve it from a registry.
func appImageRef(name string) string {
	return fmt.Sprintf("localhost/%s:latest", name)
}

// ensureContainerImage fails a start before systemd does when the image a container workload runs is
// not built yet, so the operator is pointed at the command that builds it instead of a restart loop
// whose journal only shows podman trying to reach a registry named localhost. A podman that cannot
// be queried is left for systemd to report, rather than turning an inspection failure into a start
// failure of its own.
func ensureContainerImage(ctx context.Context, cfg *storage.AppConfig) error {
	if !cfg.UsesPodManifest() {
		return nil
	}
	ref := appImageRef(cfg.Name)
	exists, err := podman.ImageExists(ctx, ref)
	if err != nil || exists {
		return nil
	}
	return fmt.Errorf("container image %s is not built yet — run `gare deploy %s` to build and start it", ref, cfg.Name)
}
