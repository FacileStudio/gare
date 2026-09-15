package storage

import (
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// StaticSection holds static site configuration.
type StaticSection struct {
	Dir  string `yaml:"dir"`
	Root string `yaml:"root"`
}

// BuildSection holds build command and containerfile configuration.
type BuildSection struct {
	Command       string `yaml:"command"`
	Containerfile string `yaml:"containerfile"`
	Context       string `yaml:"context"`
}

// GareFile represents the configuration parsed from gare.yaml or gare.yml.
type GareFile struct {
	Type          string         `yaml:"type"`
	Containerfile string         `yaml:"containerfile"`
	Context       string         `yaml:"context"`
	StaticDir     string         `yaml:"static_dir"`
	BuildCmd      string         `yaml:"build_cmd"`
	Static        *StaticSection `yaml:"static"`
	Build         *BuildSection  `yaml:"build"`
}

// ResolveType returns the application type, defaulting to "container".
func (g *GareFile) ResolveType() string {
	if g != nil && strings.EqualFold(g.Type, "static") {
		return "static"
	}
	return "container"
}

// ResolveContainerfile returns the containerfile path if configured.
func (g *GareFile) ResolveContainerfile() string {
	if g == nil {
		return ""
	}
	if g.Build != nil && g.Build.Containerfile != "" {
		return g.Build.Containerfile
	}
	return g.Containerfile
}

// ResolveContext returns the build context directory if configured.
func (g *GareFile) ResolveContext() string {
	if g == nil {
		return ""
	}
	if g.Build != nil && g.Build.Context != "" {
		return g.Build.Context
	}
	return g.Context
}

// ResolveStaticDir returns the static directory path if configured.
func (g *GareFile) ResolveStaticDir() string {
	if g == nil {
		return ""
	}
	if g.Static != nil {
		if g.Static.Dir != "" {
			return g.Static.Dir
		}
		if g.Static.Root != "" {
			return g.Static.Root
		}
	}
	return g.StaticDir
}

// ResolveBuildCmd returns the build command if configured.
func (g *GareFile) ResolveBuildCmd() string {
	if g == nil {
		return ""
	}
	if g.Build != nil && g.Build.Command != "" {
		return g.Build.Command
	}
	return g.BuildCmd
}

// LoadGareFile reads and parses gare.yaml or gare.yml from the repo directory.
func LoadGareFile(repoDir string) (*GareFile, error) {
	filePath := filepath.Join(repoDir, "gare.yaml")
	data, err := os.ReadFile(filePath)
	if os.IsNotExist(err) {
		filePath = filepath.Join(repoDir, "gare.yml")
		data, err = os.ReadFile(filePath)
		if os.IsNotExist(err) {
			return nil, nil
		}
	}
	if err != nil {
		return nil, err
	}
	var gf GareFile
	if err := yaml.Unmarshal(data, &gf); err != nil {
		return nil, err
	}
	return &gf, nil
}

// IsStatic reports whether the application is configured as a static site.
func (c *AppConfig) IsStatic() bool {
	return c != nil && strings.EqualFold(c.AppType, "static")
}
