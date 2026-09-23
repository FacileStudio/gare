package main

import (
	"context"
	"fmt"

	"github.com/FacileStudio/gare/internal/systemd"
)

// checkPodmanWorkloads reports whether podman can run the commands gare's units spell out.
func checkPodmanWorkloads(ctx context.Context) {
	if err := systemd.CheckPodmanWorkloads(ctx); err != nil {
		printWarning(err.Error())
		fmt.Printf("\nContainer and static workloads deploy through podman kube play and podman run; compose workloads are unaffected.\n\n")
		return
	}
	printSuccess("Found podman workload runtime")
}
