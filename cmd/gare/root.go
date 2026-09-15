package main

import (
	"context"

	"charm.land/fang/v2"
	"github.com/spf13/cobra"
)

// Execute runs the root command with fang styling and version handling.
func Execute(version string) error {
	return fang.Execute(context.Background(), newRootCmd(version), fang.WithVersion(version))
}

// NewRootCmd builds the root gare command and registers subcommands.
func NewRootCmd(version string) *cobra.Command {
	return newRootCmd(version)
}

func newRootCmd(version string) *cobra.Command {
	root := &cobra.Command{
		Use:           "gare",
		Version:       version,
		SilenceErrors: true,
		Short:         "Zero-daemon deployment CLI and GitOps orchestrator for Podman + Kubernetes workloads",
		Long: `gare is a minimalist, self-contained application manager that embraces
native Linux primitives for Podman and Kubernetes workloads:
- Zero Docker: uses podman kube play natively
- Direct systemd user units: synthesized directly without Quadlet
- Caddy ingress: drop-in snippets under /etc/caddy/conf.d/
- Stateless & file-driven: state stored purely on the filesystem`,
	}

	root.SetVersionTemplate("{{.Name}} {{.Version}}\n")

	root.AddCommand(NewInitCmd())
	root.AddCommand(NewAppCmd())
	root.AddCommand(NewDeployCmd())
	root.AddCommand(NewListCmd())
	root.AddCommand(NewLogsCmd())
	root.AddCommand(NewDestroyCmd())
	root.AddCommand(NewServerCmd())

	return root
}
