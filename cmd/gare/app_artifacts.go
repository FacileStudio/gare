package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/FacileStudio/gare/internal/atomicfile"
	"github.com/FacileStudio/gare/internal/storage"
	"github.com/FacileStudio/gare/internal/systemd"
)

func writeAppArtifacts(name, appDir string, opts appCreateOptions) error {
	manifestPath := storage.GetManifestPath(appDir)
	if err := resolveManifest(name, appDir, manifestPath, opts.port, opts.containerPort); err != nil {
		return err
	}
	if err := systemd.WriteUnit(name, manifestPath); err != nil {
		return fmt.Errorf("failed to write systemd unit: %w", err)
	}
	return saveAppMetadata(name, appDir, opts)
}

func mergeGareFileDefaults(opts appCreateOptions, gf *storage.GareFile) (appCreateOptions, error) {
	if gf == nil {
		return opts, nil
	}
	workload, err := gf.ResolveWorkload()
	if err != nil {
		return opts, err
	}
	opts.appType = defaultStr(opts.appType, string(workload))
	opts.composeFile = defaultStr(opts.composeFile, gf.ResolveComposeFile())
	opts.containerfile = defaultStr(opts.containerfile, gf.ResolveContainerfile())
	opts.contextDir = defaultStr(opts.contextDir, gf.ResolveContext())
	opts.staticDir = defaultStr(opts.staticDir, gf.ResolveStaticDir())
	opts.buildCmd = defaultStr(opts.buildCmd, gf.ResolveBuildCmd())
	opts.healthcheck = defaultStr(opts.healthcheck, gf.ResolveHealthcheck())
	if len(opts.healthProbes) == 0 {
		opts.healthProbes = gf.ResolveHealthProbes()
	}
	if opts.port == 0 {
		opts.port = gf.ResolvePort()
	}
	if opts.containerPort == 0 {
		opts.containerPort = gf.ResolveContainerPort()
	}
	if len(opts.tags) == 0 {
		opts.tags = gf.ResolveTags()
	}
	return opts, nil
}

func writeStaticArtifacts(name, appDir string, opts appCreateOptions) error {
	repoDir := storage.GetRepoDir(appDir)
	staticPath := filepath.Join(repoDir, opts.staticDir)
	if err := systemd.WriteStaticUnit(name, opts.port, staticPath); err != nil {
		return fmt.Errorf("failed to write systemd unit: %w", err)
	}
	opts.appType = "static"
	return saveAppMetadata(name, appDir, opts)
}

func resolveManifest(name, appDir, manifestPath string, port int, containerPort int) error {
	repoManifest := filepath.Join(storage.GetRepoDir(appDir), "manifest.yaml")
	if data, err := os.ReadFile(repoManifest); err == nil {
		if err := atomicfile.WriteFile(manifestPath, data, 0644); err != nil {
			return fmt.Errorf("failed to write manifest: %w", err)
		}
		return nil
	}
	if containerPort <= 0 {
		containerPort = port
	}
	return storage.GenerateDefaultManifest(name, containerPort, port, manifestPath)
}

func saveAppMetadata(name, appDir string, opts appCreateOptions) error {
	appCfg := &storage.AppConfig{
		Name:          name,
		RepoURL:       opts.repo,
		Port:          opts.port,
		ContainerPort: opts.containerPort,
		Branch:        opts.branch,
		CreatedAt:     time.Now().UTC().Format(time.RFC3339),
		AppType:       opts.appType,
		ComposeFile:   opts.composeFile,
		Containerfile: opts.containerfile,
		ContextDir:    opts.contextDir,
		StaticDir:     opts.staticDir,
		BuildCmd:      opts.buildCmd,
	}
	appCfg.SetHealth(opts.healthcheck, opts.healthProbes)
	if err := appCfg.AddTags(opts.tags...); err != nil {
		return fmt.Errorf("failed to add tags: %w", err)
	}
	if err := storage.SaveConfig(appDir, appCfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}
	printSuccess(fmt.Sprintf("App %q successfully created on port %d", name, opts.port))
	return nil
}

func syncGareFileConfig(baseDir, repoDir string, cfg *storage.AppConfig) error {
	gf, err := storage.LoadGareFile(repoDir)
	if err != nil || gf == nil {
		return err
	}
	if _, err := gf.ResolveWorkload(); err != nil {
		return err
	}
	if cfg.BuildCmd == "" {
		cfg.BuildCmd = gf.ResolveBuildCmd()
	}
	if err := storage.ValidateHealthProbes(gf.ResolveHealthProbes()); err != nil {
		return err
	}
	applyGareTags(cfg, gf)
	if err := syncGareFilePorts(baseDir, cfg, gf); err != nil {
		return err
	}
	applyGareWorkloadConfig(cfg, gf)
	applyGareHealthConfig(repoDir, cfg, gf)
	return nil
}

func applyGareTags(cfg *storage.AppConfig, gf *storage.GareFile) {
	tags := gf.ResolveTags()
	if tags == nil {
		return
	}
	temp := &storage.AppConfig{}
	if err := temp.AddTags(tags...); err != nil {
		printWarning(fmt.Sprintf("Invalid tags in gare.yml: %v", err))
		return
	}
	cfg.Tags = temp.Tags
}

