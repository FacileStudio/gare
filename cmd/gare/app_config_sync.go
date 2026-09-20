package main

import (
	"fmt"
	"path/filepath"

	"github.com/FacileStudio/gare/internal/storage"
)

func syncGareFilePorts(baseDir string, cfg *storage.AppConfig, gf *storage.GareFile) error {
	reqPort := gf.ResolvePort()
	if reqPort > 0 && reqPort != cfg.Port {
		if _, err := storage.DiscoverAvailablePort(baseDir, reqPort); err != nil {
			return fmt.Errorf("port %d in gare.yml is not available — choose another port: %w", reqPort, err)
		}
		cfg.Port = reqPort
	}
	if reqCPort := gf.ResolveContainerPort(); reqCPort > 0 {
		cfg.ContainerPort = reqCPort
	}
	return nil
}

func applyGareWorkloadConfig(cfg *storage.AppConfig, gf *storage.GareFile) {
	applyGareWorkloadType(cfg, gf)
	if cfg.ComposeFile == "" {
		cfg.ComposeFile = gf.ResolveComposeFile()
	}
	if cfg.IsStatic() {
		applyGareStaticConfig(cfg, gf)
		return
	}
	if cfg.UsesPodManifest() {
		applyGareContainerConfig(cfg, gf)
	}
}

func applyGareHealthConfig(repoDir string, cfg *storage.AppConfig, gf *storage.GareFile) {
	path := cfg.HealthPath()
	if path == "" {
		path = gf.ResolveHealthcheck()
	}
	probes := gf.ResolveHealthProbes()
	if len(probes) == 0 {
		probes = deriveComposeProbes(repoDir, cfg)
	}
	cfg.SetHealth(path, probes)
}

func deriveComposeProbes(repoDir string, cfg *storage.AppConfig) []storage.HealthProbe {
	if !cfg.IsCompose() {
		return nil
	}
	composeFile, err := storage.LocateComposeFile(repoDir, cfg.ComposeFile)
	if err != nil {
		return nil
	}
	probes, err := storage.ComposeHealthProbes(filepath.Join(repoDir, composeFile))
	if err != nil || len(probes) == 0 {
		return nil
	}
	printInfo(fmt.Sprintf("Derived %d health probe(s) from %s: %s", len(probes), composeFile, probeSummary(probes)))
	return probes
}

func applyGareWorkloadType(cfg *storage.AppConfig, gf *storage.GareFile) {
	workload, err := gf.ResolveWorkload()
	if err == nil && gf.Type != "" {
		cfg.AppType = string(workload)
	}
}

func applyGareStaticConfig(cfg *storage.AppConfig, gf *storage.GareFile) {
	dir := gf.ResolveStaticDir()
	if dir != "" && (cfg.StaticDir == "" || cfg.StaticDir == ".") {
		cfg.StaticDir = dir
	}
}

func applyGareContainerConfig(cfg *storage.AppConfig, gf *storage.GareFile) {
	if cfg.Containerfile == "" {
		cfg.Containerfile = gf.ResolveContainerfile()
	}
	contextDir := gf.ResolveContext()
	if contextDir != "" && (cfg.ContextDir == "" || cfg.ContextDir == ".") {
		cfg.ContextDir = contextDir
	}
}
