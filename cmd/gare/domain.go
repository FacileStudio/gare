package main

import (
	"context"
	"time"

	"github.com/spf13/cobra"
)

type domainListOptions struct {
	jsonOutput bool
}

// NewDomainCmd builds the domain command group.
func NewDomainCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "domain",
		Short: "Manage application domains and ingress hostnames",
	}
	cmd.AddCommand(newDomainAddCmd())
	cmd.AddCommand(newDomainRemoveCmd())
	cmd.AddCommand(newDomainListCmd())
	return cmd
}

func newDomainAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add <app> <hostname>",
		Short: "Add a domain to an application",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()
			return runDomainAdd(ctx, args[0], args[1])
		},
	}
}

func newDomainRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "remove <app> <hostname>",
		Aliases: []string{"rm", "delete"},
		Short:   "Remove a domain from an application",
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()
			return runDomainRemove(ctx, args[0], args[1])
		},
	}
}

func newDomainListCmd() *cobra.Command {
	opts := domainListOptions{}
	cmd := &cobra.Command{
		Use:     "list [<app>]",
		Aliases: []string{"ls"},
		Short:   "List configured domains",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			targetApp := ""
			if len(args) > 0 {
				targetApp = args[0]
			}
			return runDomainList(cmd.OutOrStdout(), targetApp, opts)
		},
	}
	cmd.Flags().BoolVar(&opts.jsonOutput, "json", false, "Output JSON array")
	return cmd
}
