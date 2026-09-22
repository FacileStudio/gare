package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/FacileStudio/gare/internal/builder"
	"github.com/FacileStudio/gare/internal/storage"
	"github.com/FacileStudio/gare/internal/systemd"
)

// resolveComposeWorkload resolves the compose file to run and validates the routed port.
func resolveComposeWorkload(repoDir, configured string, port int) (string, error) {
	composeFile, err := storage.LocateComposeFile(repoDir, configured)
	if err != nil {
		return "", err
	}
	if err := storage.ValidateComposePort(filepath.Join(repoDir, composeFile), port); err != nil {
		return "", err
	}
	return composeFile, nil
}

// composeProjectName derives a stable per-app compose project name.
func composeProjectName(name string) string {
	return strings.ToLower(name)
}

func writeComposeUnit(name, appDir, repoDir, composeFile string) error {
	unitData := systemd.ComposeUnitData{
		Name:        name,
		RepoDir:     repoDir,
		ComposeFile: composeFile,
		ProjectName: composeProjectName(name),
		EnvFile:     storage.GetAppEnvPath(appDir),
	}
	if err := systemd.WriteComposeUnit(unitData); err != nil {
		return fmt.Errorf("failed to write systemd unit: %w", err)
	}
	return nil
}

func writeComposeArtifacts(name, appDir string, opts appCreateOptions) error {
	repoDir := storage.GetRepoDir(appDir)
	composeFile, err := resolveComposeWorkload(repoDir, opts.composeFile, opts.port)
	if err != nil {
		return err
	}
	opts.appType = string(storage.WorkloadCompose)
	opts.composeFile = composeFile
	if err := writeComposeUnit(name, appDir, repoDir, composeFile); err != nil {
		return err
	}
	return saveAppMetadata(name, appDir, opts)
}

func prepareComposeDeploy(ctx context.Context, name, appDir, repoDir string, cfg *storage.AppConfig) error {
	composeFile, err := resolveComposeWorkload(repoDir, cfg.ComposeFile, cfg.Port)
	if err != nil {
		return err
	}
	if err := builder.CheckComposeProvider(ctx); err != nil {
		return err
	}
	if cfg.BuildCmd != "" {
		printInfo(fmt.Sprintf("Running build command: %s", cfg.BuildCmd))
		if err := builder.RunBuildCommand(ctx, repoDir, cfg.BuildCmd, os.Stdout, os.Stderr); err != nil {
			return fmt.Errorf("build command failed: %w", err)
		}
	}
	return writeComposeUnit(name, appDir, repoDir, composeFile)
}

func deployComposeApp(ctx context.Context, name, appDir, repoDir string, cfg *storage.AppConfig) error {
	if err := prepareComposeDeploy(ctx, name, appDir, repoDir, cfg); err != nil {
		return err
	}

	updateContainerIngress(cfg)

	if err := restartAppServices(ctx, cfg); err != nil {
		return err
	}

	if err := verifyHealth(ctx, cfg); err != nil {
		return err
	}

	cleanupAppDeploy(ctx)
	commitHash, _ := builder.GetCommitHash(ctx, repoDir)
	if commitHash == "" {
		commitHash = "-"
	}
	printSuccess(fmt.Sprintf("Successfully deployed compose app %s (%s) -> %s", name, commitHash, formatDeployTarget(cfg)))
	return nil
}

func destroyComposeWorkload(ctx context.Context, appDir string, cfg *storage.AppConfig) {
	repoDir := storage.GetRepoDir(appDir)
	composeFile, err := storage.LocateComposeFile(repoDir, cfg.ComposeFile)
	if err != nil {
		printWarning(fmt.Sprintf("Skipping compose teardown: %v", err))
		return
	}

	printInfo(fmt.Sprintf("Removing compose stack %s including volumes...", composeFile))
	downOpts := builder.ComposeDownOptions{
		RepoDir:     repoDir,
		ComposeFile: composeFile,
		Project:     composeProjectName(cfg.Name),
		Volumes:     true,
		Stdout:      os.Stdout,
		Stderr:      os.Stderr,
	}
	if err := builder.ComposeDown(ctx, downOpts); err != nil {
		printWarning(fmt.Sprintf("podman compose down returned error: %v", err))
	}
	verifyComposeContainersGone(ctx, cfg, "teardown")
}

func warnStrayComposeFile(repoDir string, isCompose bool) {
	if isCompose {
		return
	}
	if detected := storage.DetectComposeFile(repoDir); detected != "" {
		printWarning(fmt.Sprintf("compose file %s found but the workload type is not compose; set type: compose in gare.yml to deploy it", detected))
	}
}
