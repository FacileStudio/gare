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
	if opts.domain != "" {
		if err := caddy.WriteSnippet(caddy.ResolveConfDir(), name, opts.domain, opts.port); err != nil {
			printWarning(fmt.Sprintf("Could not write Caddy snippet (%v)", err))
		}
	}
	return saveAppMetadata(name, appDir, opts)
}

func writeStaticArtifacts(name, appDir string, opts appCreateOptions) error {
	repoDir := storage.GetRepoDir(appDir)
	staticPath := filepath.Join(repoDir, opts.staticDir)
	if err := systemd.WriteStaticUnit(name, opts.port, staticPath); err != nil {
		return fmt.Errorf("failed to write systemd unit: %w", err)
	}
	if opts.domain != "" {
		if err := caddy.WriteSnippet(caddy.ResolveConfDir(), name, opts.domain, opts.port); err != nil {
			printWarning(fmt.Sprintf("Could not write Caddy snippet (%v)", err))
		}
	}
	opts.appType = "static"
	return saveAppMetadata(name, appDir, opts)
}

func resolveManifest(name, appDir, manifestPath string, port int) error {
	repoManifest := filepath.Join(storage.GetRepoDir(appDir), "manifest.yaml")
	if data, err := os.ReadFile(repoManifest); err == nil {
		if err := atomicfile.WriteFile(manifestPath, data, 0644); err != nil {
			return fmt.Errorf("failed to write manifest: %w", err)
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
		AppType:       opts.appType,
		Containerfile: opts.containerfile,
		ContextDir:    opts.contextDir,
		StaticDir:     opts.staticDir,
		BuildCmd:      opts.buildCmd,
		Healthcheck:   opts.healthcheck,
	}
	if err := storage.SaveConfig(appDir, appCfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}
	if opts.domain != "" {
		printSuccess(fmt.Sprintf("App %q successfully created on port %d (%s)", name, opts.port, opts.domain))
	} else {
		printSuccess(fmt.Sprintf("App %q successfully created on port %d", name, opts.port))
	}
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
	existingEnvs, _ := storage.GetManifestEnv(appManifest)
	if err := atomicfile.WriteFile(appManifest, data, 0644); err != nil {
		return fmt.Errorf("failed to sync manifest: %w", err)
	}
	if len(existingEnvs) > 0 {
		if err := storage.SetManifestEnv(appManifest, existingEnvs); err != nil {
			return fmt.Errorf("failed to restore manifest environment: %w", err)
		}
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
	if opts.healthcheck == "" {
		opts.healthcheck = gf.ResolveHealthcheck()
	}
	if opts.port == 0 {
		opts.port = gf.ResolvePort()
	}
	if opts.domain == "" {
		opts.domain = gf.ResolveDomain()
	}
	return opts
}

func syncGareFileConfig(baseDir, repoDir string, cfg *storage.AppConfig) error {
	gf, err := storage.LoadGareFile(repoDir)
	if err != nil || gf == nil {
		return err
	}
	if cfg.BuildCmd == "" {
		cfg.BuildCmd = gf.ResolveBuildCmd()
	}
	if cfg.Healthcheck == "" {
		cfg.Healthcheck = gf.ResolveHealthcheck()
	}
	if cfg.Domain == "" {
		cfg.Domain = gf.ResolveDomain()
	}
	reqPort := gf.ResolvePort()
	if reqPort > 0 && reqPort != cfg.Port {
		if _, err := storage.DiscoverAvailablePort(baseDir, reqPort); err != nil {
			return fmt.Errorf("port %d in gare.yml is not available: %w", reqPort, err)
		}
		cfg.Port = reqPort
	}
	applyGareWorkloadConfig(cfg, gf)
	return nil
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
