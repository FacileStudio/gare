package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/FacileStudio/gare/internal/atomicfile"
	"github.com/FacileStudio/gare/internal/caddy"
	"github.com/FacileStudio/gare/internal/storage"
	"github.com/FacileStudio/gare/internal/systemd"
)

func writeAppArtifacts(name, appDir string, opts appCreateOptions) error {
	manifestPath := storage.GetManifestPath(appDir)
	if err := resolveManifest(name, appDir, manifestPath, opts.port); err != nil {
		return err
	}
	if err := systemd.WriteUnit(name, manifestPath); err != nil {
		return fmt.Errorf("failed to write systemd unit: %w", err)
	}
	if err := caddy.WriteSnippet(caddy.DefaultConfDir, name, opts.domain, opts.port); err != nil {
		printWarning(fmt.Sprintf("Could not write Caddy snippet (%v)", err))
	}
	return saveAppMetadata(name, appDir, opts)
}

func writeStaticArtifacts(name, appDir string, opts appCreateOptions) error {
	repoDir := storage.GetRepoDir(appDir)
	staticPath := filepath.Join(repoDir, opts.staticDir)
	if err := caddy.WriteStaticSnippet(caddy.DefaultConfDir, name, opts.domain, staticPath); err != nil {
		printWarning(fmt.Sprintf("Could not write Caddy snippet (%v)", err))
	}
	appCfg := &storage.AppConfig{
		Name:      name,
		RepoURL:   opts.repo,
		Domain:    opts.domain,
		Port:      0,
		Branch:    opts.branch,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		AppType:   "static",
		StaticDir: opts.staticDir,
		BuildCmd:  opts.buildCmd,
	}
	if err := storage.SaveConfig(appDir, appCfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}
	printSuccess(fmt.Sprintf("App %q successfully created as static site (%s)", name, opts.domain))
	return nil
}

func resolveManifest(name, appDir, manifestPath string, port int) error {
	repoManifest := filepath.Join(storage.GetRepoDir(appDir), "manifest.yaml")
	if data, err := os.ReadFile(repoManifest); err == nil {
		if err := atomicfile.WriteFile(manifestPath, data, 0644); err != nil {
			return fmt.Errorf("failed to copy manifest: %w", err)
		}
		return nil
	}
	return storage.GenerateDefaultManifest(name, port, manifestPath)
}

func saveAppMetadata(name, appDir string, opts appCreateOptions) error {
	appCfg := &storage.AppConfig{
		Name:          name,
		RepoURL:       opts.repo,
		Domain:        opts.domain,
		Port:          opts.port,
		Branch:        opts.branch,
		CreatedAt:     time.Now().UTC().Format(time.RFC3339),
		AppType:       "container",
		Containerfile: opts.containerfile,
		ContextDir:    opts.contextDir,
		BuildCmd:      opts.buildCmd,
	}
	if err := storage.SaveConfig(appDir, appCfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}
	printSuccess(fmt.Sprintf("App %q successfully created on port %d (%s)", name, opts.port, opts.domain))
	return nil
}

func syncRepoManifest(appDir, repoDir string) error {
	repoManifest := filepath.Join(repoDir, "manifest.yaml")
	data, err := os.ReadFile(repoManifest)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to read repo manifest: %w", err)
	}
	appManifest := storage.GetManifestPath(appDir)
	if err := atomicfile.WriteFile(appManifest, data, 0644); err != nil {
		return fmt.Errorf("failed to sync manifest: %w", err)
	}
	printSuccess("Synced manifest.yaml from repository")
	return nil
}

func mergeGareFileDefaults(opts appCreateOptions, gf *storage.GareFile) appCreateOptions {
	if gf == nil {
		return opts
	}
	if opts.appType == "" {
		opts.appType = gf.ResolveType()
	}
	if opts.containerfile == "" {
		opts.containerfile = gf.ResolveContainerfile()
	}
	if opts.contextDir == "" {
		opts.contextDir = gf.ResolveContext()
	}
	if opts.staticDir == "" {
		opts.staticDir = gf.ResolveStaticDir()
	}
	if opts.buildCmd == "" {
		opts.buildCmd = gf.ResolveBuildCmd()
	}
	return opts
}

func syncGareFileConfig(repoDir string, cfg *storage.AppConfig) {
	gf, err := storage.LoadGareFile(repoDir)
	if err != nil || gf == nil {
		return
	}
	if cfg.BuildCmd == "" {
		cfg.BuildCmd = gf.ResolveBuildCmd()
	}
	applyGareWorkloadConfig(cfg, gf)
}

func applyGareWorkloadConfig(cfg *storage.AppConfig, gf *storage.GareFile) {
	if cfg.IsStatic() {
		dir := gf.ResolveStaticDir()
		if dir != "" && (cfg.StaticDir == "" || cfg.StaticDir == ".") {
			cfg.StaticDir = dir
		}
		return
	}
	if cfg.Containerfile == "" {
		cfg.Containerfile = gf.ResolveContainerfile()
	}
	ctx := gf.ResolveContext()
	if ctx != "" && (cfg.ContextDir == "" || cfg.ContextDir == ".") {
		cfg.ContextDir = ctx
	}
}
