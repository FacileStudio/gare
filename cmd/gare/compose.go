package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/FacileStudio/gare/internal/compose"
	"github.com/FacileStudio/gare/internal/podman"
	"github.com/FacileStudio/gare/internal/storage"
	"github.com/FacileStudio/gare/internal/systemd"
)

// resolveComposeWorkload resolves the compose file to run and validates the routed port.
func resolveComposeWorkload(repoDir, configured string, port int) (string, error) {
	composeFile, err := compose.LocateFile(repoDir, configured)
	if err != nil {
		return "", err
	}
	if err := compose.ValidatePort(filepath.Join(repoDir, composeFile), port); err != nil {
		return "", err
	}
	return composeFile, nil
}

// composeProjectName derives a stable per-app compose project name.
func composeProjectName(name string) string {
	return strings.ToLower(name)
}

// writeComposeUnit writes the synthesized unit for a compose stack and hands the stack over to it.
// Its guard checks the unit file without probing the podman workload runtime the container and
// static units need, because a compose unit runs through the podman compose provider rather than
// kube play, and it retires the Quadlet sources a previous workload type left behind so no stale
// source keeps generating a unit of the same name.
func writeComposeUnit(ctx context.Context, name, appDir, repoDir, composeFile string) error {
	previous, err := guardUnitFile(name)
	if err != nil {
		return err
	}
	unitData := systemd.ComposeUnitData{
		Name:        name,
		RepoDir:     repoDir,
		ComposeFile: composeFile,
		ProjectName: composeProjectName(name),
		EnvFile:     storage.GetAppEnvPath(appDir),
	}
	return applyUnitWrite(ctx, name, systemd.ComposeUnitDescription(name), previous,
		func() error { return systemd.WriteComposeUnit(unitData) })
}

func writeComposeArtifacts(ctx context.Context, name, appDir string, opts appCreateOptions) error {
	repoDir := storage.GetRepoDir(appDir)
	composeFile, err := resolveComposeWorkload(repoDir, opts.composeFile, opts.port)
	if err != nil {
		return err
	}
	opts.appType = string(storage.WorkloadCompose)
	opts.composeFile = composeFile
	if err := writeComposeUnit(ctx, name, appDir, repoDir, composeFile); err != nil {
		return err
	}
	return saveAppMetadata(name, appDir, opts)
}

// prepareComposeDeploy resolves the compose file, checks the provider and the routed port, runs the
// build command, then writes the synthesized unit the stack is started and torn down through.
func prepareComposeDeploy(ctx context.Context, name, appDir, repoDir string, cfg *storage.AppConfig) error {
	composeFile, err := resolveComposeWorkload(repoDir, cfg.ComposeFile, cfg.Port)
	if err != nil {
		return err
	}
	if err := podman.CheckComposeProvider(ctx); err != nil {
		return err
	}
	if err := runBuildCommand(ctx, repoDir, cfg.BuildCmd); err != nil {
		return err
	}
	return writeComposeUnit(ctx, name, appDir, repoDir, composeFile)
}

func destroyComposeWorkload(ctx context.Context, appDir string, cfg *storage.AppConfig) {
	repoDir := storage.GetRepoDir(appDir)
	composeFile, err := compose.LocateFile(repoDir, cfg.ComposeFile)
	if err != nil {
		printWarning(fmt.Sprintf("Skipping compose teardown: %v", err))
		return
	}

	printInfo(fmt.Sprintf("Removing compose stack %s including volumes...", composeFile))
	downOpts := podman.ComposeDownOptions{
		RepoDir:     repoDir,
		ComposeFile: composeFile,
		Project:     composeProjectName(cfg.Name),
		Volumes:     true,
		Stdout:      os.Stdout,
		Stderr:      os.Stderr,
	}
	if err := podman.ComposeDown(ctx, downOpts); err != nil {
		printWarning(fmt.Sprintf("podman compose down returned error: %v", err))
	}
	verifyComposeContainersGone(ctx, cfg, "teardown")
}

func warnStrayComposeFile(repoDir string, isCompose bool) {
	if isCompose {
		return
	}
	if detected := compose.DetectFile(repoDir); detected != "" {
		printWarning(fmt.Sprintf("compose file %s found but the workload type is not compose; set type: compose in gare.yml to deploy it", detected))
	}
}
