package main

import (
	"context"
	"fmt"
	"time"

	"github.com/FacileStudio/gare/internal/dotenv"
	"github.com/FacileStudio/gare/internal/manifest"
	"github.com/FacileStudio/gare/internal/storage"
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
			return runEnvList(cmd.OutOrStdout(), args[0], opts)
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

func runEnvSet(ctx context.Context, name string, assignments []string) error {
	vars, err := dotenv.ParseAssignments(assignments)
	if err != nil {
		return err
	}
	store, err := applyEnvChange(name,
		func(appDir string) error { return storage.SetAppEnv(appDir, vars) },
		func(manifestPath string) error { return manifest.SetEnv(manifestPath, vars) },
	)
	if err != nil {
		return fmt.Errorf("failed to set environment variables: %w", err)
	}
	printSuccess(fmt.Sprintf("Updated %d environment variable(s) in %s", len(vars), store))
	return reloadIfActive(ctx, name)
}

func runEnvUnset(ctx context.Context, name string, keys []string) error {
	store, err := applyEnvChange(name,
		func(appDir string) error { return storage.UnsetAppEnv(appDir, keys) },
		func(manifestPath string) error { return manifest.UnsetEnv(manifestPath, keys) },
	)
	if err != nil {
		return fmt.Errorf("failed to unset environment variables: %w", err)
	}
	printSuccess(fmt.Sprintf("Removed %d environment variable(s) from %s", len(keys), store))
	return reloadIfActive(ctx, name)
}

func runEnvLoad(ctx context.Context, name string, opts envLoadOptions) error {
	vars, err := dotenv.LoadFile(opts.envFile)
	if err != nil {
		return fmt.Errorf("failed to read env file: %w", err)
	}
	if _, err := applyEnvChange(name,
		func(appDir string) error { return storage.SetAppEnv(appDir, vars) },
		func(manifestPath string) error { return manifest.SetEnv(manifestPath, vars) },
	); err != nil {
		return fmt.Errorf("failed to load environment variables: %w", err)
	}
	printSuccess(fmt.Sprintf("Loaded %d environment variable(s) from %s", len(vars), opts.envFile))
	return reloadIfActive(ctx, name)
}
