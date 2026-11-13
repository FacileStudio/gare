package storage

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/FacileStudio/gare/apps/api/internal/health"
	"github.com/FacileStudio/gare/apps/api/internal/repopath"
	"gopkg.in/yaml.v3"
)

// GareFile represents the configuration parsed from gare.yaml or gare.yml.
type GareFile struct {
	Type          string         `yaml:"type"`
	ComposeFile   string         `yaml:"compose_file,omitempty"`
	Service       string         `yaml:"service,omitempty"`
	Port          int            `yaml:"port"`
	ContainerPort int            `yaml:"container_port,omitempty"`
	Containerfile string         `yaml:"containerfile"`
	Context       string         `yaml:"context"`
	StaticDir     string         `yaml:"static_dir"`
	BuildCmd      string         `yaml:"build_cmd"`
	Healthcheck   string         `yaml:"healthcheck"`
	Healthchecks  []health.Probe `yaml:"healthchecks,omitempty"`
	Tags          []string       `yaml:"tags,omitempty"`
}

// ResolveComposeFile returns the configured compose file name, trimmed of surrounding whitespace.
func (g *GareFile) ResolveComposeFile() string {
	if g == nil {
		return ""
	}
	return strings.TrimSpace(g.ComposeFile)
}

// ValidatePaths rejects a static_dir, context or containerfile that would reach outside the
// repository, since a repository controls its own gare.yml and must not be able to point gare at
// another app's directory. A nil file or an unset key has nothing to check.
func (g *GareFile) ValidatePaths() error {
	if g == nil {
		return nil
	}
	return repopath.ValidateSet(map[string]string{
		"static_dir":    g.StaticDir,
		"context":       g.Context,
		"containerfile": g.Containerfile,
	})
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
