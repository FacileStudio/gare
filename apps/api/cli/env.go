package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/FacileStudio/gare/apps/api/internal/dotenv"
	"github.com/spf13/cobra"
)

type envListOptions struct {
	jsonOutput bool
}

type envLoadOptions struct {
	envFile string
}

// NewEnvCmd builds the env command group.
func NewEnvCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "env",
		Short: "Manage application environment variables",
	}
	cmd.AddCommand(newEnvSetCmd())
	cmd.AddCommand(newEnvUnsetCmd())
	cmd.AddCommand(newEnvListCmd())
	cmd.AddCommand(newEnvLoadCmd())
	return cmd
}

func newEnvSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set <app> KEY=VALUE [KEY2=VALUE2...]",
		Short: "Set environment variables for an application",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()
			return runEnvSet(ctx, args[0], args[1:])
		},
	}
}

func newEnvUnsetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "unset <app> KEY [KEY2...]",
		Short: "Unset environment variables for an application",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()
			return runEnvUnset(ctx, args[0], args[1:])
		},
	}
}

func newEnvListCmd() *cobra.Command {
	opts := envListOptions{}
	cmd := &cobra.Command{
		Use:   "list <app>",
		Short: "List environment variables for an application",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runEnvList(cmd.Context(), cmd.OutOrStdout(), args[0], opts)
		},
	}
	cmd.Flags().BoolVar(&opts.jsonOutput, "json", false, "Output JSON object")
	return cmd
}

func newEnvLoadCmd() *cobra.Command {
	opts := envLoadOptions{}
	cmd := &cobra.Command{
		Use:   "load <app> -f <env-file>",
		Short: "Load environment variables from a file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()
			return runEnvLoad(ctx, args[0], opts)
		},
	}
	cmd.Flags().StringVarP(&opts.envFile, "file", "f", "", "Path to .env file")
	if err := cmd.MarkFlagRequired("file"); err != nil {
		return cmd
	}
	return cmd
}

// runEnvSet parses KEY=VALUE assignments and sets them through the engine.
func runEnvSet(ctx context.Context, name string, assignments []string) error {
	vars, err := dotenv.ParseAssignments(assignments)
	if err != nil {
		return err
	}
	result, err := newEngine(ctx).SetEnv(ctx, name, vars)
	return reportEnvChange(name, result, err)
}

// runEnvUnset removes the named keys through the engine.
func runEnvUnset(ctx context.Context, name string, keys []string) error {
	result, err := newEngine(ctx).UnsetEnv(ctx, name, keys)
	return reportEnvChange(name, result, err)
}

// runEnvLoad reads an env file and sets its variables through the engine.
func runEnvLoad(ctx context.Context, name string, opts envLoadOptions) error {
	vars, err := dotenv.LoadFile(opts.envFile)
	if err != nil {
		return fmt.Errorf("failed to read env file: %w", err)
	}
	result, err := newEngine(ctx).LoadEnv(ctx, name, vars, opts.envFile)
	return reportEnvChange(name, result, err)
}
