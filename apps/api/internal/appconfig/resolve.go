// Package appconfig decides the configuration an application runs with, from the three places a
// setting can come from, so create and deploy cannot disagree about which one wins.
package appconfig

import (
	"fmt"
	"strings"

	"github.com/FacileStudio/gare/apps/api/internal/health"
	"github.com/FacileStudio/gare/apps/api/internal/storage"
)

// Flags holds the settings the operator passed on the command line. Every flag defaults to its zero
// value, so a zero field, or a nil Tags, is a flag the operator left alone and never outranks
// gare.yml.
type Flags struct {
	Type          string
	ComposeFile   string
	Containerfile string
	Context       string
	StaticDir     string
	BuildCmd      string
	Healthcheck   string
	Port          int
	ContainerPort int
	Tags          []string
}

// fileLayer is what gare.yml sets, in the shape of the flags plus the probes no flag can declare.
type fileLayer struct {
	Flags
	probes []health.Probe
}

// Resolve returns the configuration an application runs with. For every key the precedence is the
// same: a flag the operator actually passed wins, then gare.yml, then the value stored in
// config.json, then the default. So a value changed in gare.yml takes effect at the next deploy,
// and a key removed from gare.yml keeps its last stored value. The defaults are a container
// workload, or a static one when only a static directory is known, served from the repository root.
// Resolve reads no file and changes neither stored nor gareFile, which may both be nil; it rejects a
// gare.yml with an unknown type, a path outside the repository, an invalid probe or an invalid tag.
func Resolve(flags Flags, gareFile *storage.GareFile, stored *storage.AppConfig) (*storage.AppConfig, error) {
	file, err := readGareFile(gareFile)
	if err != nil {
		return nil, err
	}
	cfg := &storage.AppConfig{}
	if stored != nil {
		*cfg = *stored
	}
	cfg.AppType = first(strings.ToLower(strings.TrimSpace(flags.Type)), file.Type, cfg.AppType)
	cfg.ComposeFile = first(flags.ComposeFile, file.ComposeFile, cfg.ComposeFile)
	cfg.Containerfile = first(flags.Containerfile, file.Containerfile, cfg.Containerfile)
	cfg.ContextDir = first(flags.Context, file.Context, cfg.ContextDir)
	cfg.StaticDir = first(flags.StaticDir, file.StaticDir, cfg.StaticDir)
	cfg.BuildCmd = first(flags.BuildCmd, file.BuildCmd, cfg.BuildCmd)
	cfg.Port = first(flags.Port, file.Port, cfg.Port)
	cfg.ContainerPort = first(flags.ContainerPort, file.ContainerPort, cfg.ContainerPort)
	cfg.SetHealth(first(flags.Healthcheck, file.Healthcheck, cfg.HealthPath()), firstProbes(file.probes, storedProbes(cfg)))
	tags, err := resolveTags(flags.Tags, file.Tags, cfg.Tags)
	if err != nil {
		return nil, err
	}
	cfg.Tags = tags
	return cfg, applyDefaults(cfg)
}

// readGareFile validates gare.yml and returns what it sets. A missing file sets nothing.
func readGareFile(gf *storage.GareFile) (fileLayer, error) {
	if gf == nil {
		return fileLayer{}, nil
	}
	workload, err := gf.ResolveWorkload()
	if err != nil {
		return fileLayer{}, err
	}
	if err := gf.ValidatePaths(); err != nil {
		return fileLayer{}, fmt.Errorf("gare.yml: %w", err)
	}
	if err := storage.ValidateHealthProbes(gf.Healthchecks); err != nil {
		return fileLayer{}, err
	}
	layer := fileLayer{probes: gf.Healthchecks, Flags: Flags{
		ComposeFile: gf.ResolveComposeFile(), Containerfile: gf.Containerfile, Context: gf.Context,
		StaticDir: gf.StaticDir, BuildCmd: gf.BuildCmd, Healthcheck: gf.Healthcheck,
		Port: gf.Port, ContainerPort: gf.ContainerPort, Tags: gf.Tags,
	}}
	if strings.TrimSpace(gf.Type) != "" {
		layer.Type = string(workload)
	}
	return layer, nil
}

// first returns the first value that is set, in precedence order.
func first[T comparable](values ...T) T {
	var zero T
	for _, value := range values {
		if value != zero {
			return value
		}
	}
	return zero
}

// firstProbes returns the first probe list that declares any probe.
func firstProbes(lists ...[]health.Probe) []health.Probe {
	for _, probes := range lists {
		if len(probes) > 0 {
			return probes
		}
	}
	return nil
}

// storedProbes returns the probes config.json declares, leaving out the default probe HealthProbes
// derives from the port, so the default is never frozen into an explicit setting.
func storedProbes(cfg *storage.AppConfig) []health.Probe {
	if cfg.Health == nil {
		return nil
	}
	return cfg.Health.Probes
}

// resolveTags picks the tag list by the same precedence and normalises it. A list is set when it is
// not nil, so `tags: []` in gare.yml clears the stored tags.
func resolveTags(flagTags, fileTags, stored []string) ([]string, error) {
	tags := stored
	if fileTags != nil {
		tags = fileTags
	}
	if flagTags != nil {
		tags = flagTags
	}
	normalized := &storage.AppConfig{}
	if err := normalized.AddTags(tags...); err != nil {
		return nil, fmt.Errorf("invalid tags: %w", err)
	}
	return normalized.Tags, nil
}

// applyDefaults fills what no layer set and rejects a workload type gare cannot run.
func applyDefaults(cfg *storage.AppConfig) error {
	if cfg.AppType == "" && cfg.StaticDir != "" {
		cfg.AppType = string(storage.WorkloadStatic)
	}
	workload, err := cfg.ResolveWorkloadType()
	if err != nil {
		return err
	}
	cfg.AppType = string(workload)
	if workload == storage.WorkloadStatic && cfg.StaticDir == "" {
		cfg.StaticDir = "."
	}
	return nil
}
