package main

import (
	"github.com/spf13/cobra"
)

type tagListOptions struct {
	jsonOutput bool
}

// NewTagCmd builds the tag command group.
func NewTagCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "tag",
		Aliases: []string{"tags"},
		Short:   "Manage application tags",
	}
	cmd.AddCommand(newTagAddCmd())
	cmd.AddCommand(newTagRemoveCmd())
	cmd.AddCommand(newTagListCmd())
	return cmd
}

func newTagAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add <app> <tag...>",
		Short: "Add one or more tags to an application",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTagAdd(args[0], args[1:])
		},
	}
}

func newTagRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "remove <app> <tag...>",
		Aliases: []string{"rm", "delete"},
		Short:   "Remove one or more tags from an application",
		Args:    cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTagRemove(args[0], args[1:])
		},
	}
}

func newTagListCmd() *cobra.Command {
	opts := tagListOptions{}
	cmd := &cobra.Command{
		Use:     "list [<app>]",
		Aliases: []string{"ls"},
		Short:   "List configured tags",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			targetApp := ""
			if len(args) > 0 {
				targetApp = args[0]
			}
			return runTagList(cmd.OutOrStdout(), targetApp, opts)
		},
	}
	cmd.Flags().BoolVar(&opts.jsonOutput, "json", false, "Output JSON array")
	return cmd
}
