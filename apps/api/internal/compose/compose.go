// Package compose reads the compose file a compose workload hands to podman compose.
package compose

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/FacileStudio/gare/apps/api/internal/repopath"
	"gopkg.in/yaml.v3"
)

var fileCandidates = []string{"compose.yml", "compose.yaml", "docker-compose.yml", "docker-compose.yaml"}

type document struct {
	Services map[string]service  `yaml:"services"`
	Networks map[string]*network `yaml:"networks"`
}

type service struct {
	Ports       []any        `yaml:"ports"`
	Healthcheck *healthcheck `yaml:"healthcheck"`
}

type healthcheck struct {
	Test any `yaml:"test"`
}

// DetectFile returns the name of the first compose file found in repoDir.
func DetectFile(repoDir string) string {
	for _, name := range fileCandidates {
		info, err := os.Stat(filepath.Join(repoDir, name))
		if err == nil && !info.IsDir() {
			return name
		}
	}
	return ""
}

// LocateFile returns the compose file to run, defaulting to discovery in repoDir.
func LocateFile(repoDir, configured string) (string, error) {
	name := strings.TrimSpace(configured)
	if name != "" {
		if err := repopath.Validate("compose file", name); err != nil {
			return "", err
		}
		if _, err := os.Stat(filepath.Join(repoDir, name)); err != nil {
			return "", fmt.Errorf("compose file %q not found in repository: %w", name, err)
		}
		return name, nil
	}
	if detected := DetectFile(repoDir); detected != "" {
		return detected, nil
	}
	return "", fmt.Errorf("no compose file found in repository (looked for %s)", strings.Join(fileCandidates, ", "))
}

// PublishedPorts returns the host ports published by a compose file.
func PublishedPorts(composePath string) ([]int, error) {
	doc, err := readDocument(composePath)
	if err != nil {
		return nil, err
	}
	var ports []int
	for _, svc := range doc.Services {
		for _, entry := range svc.Ports {
			if published, _ := portPair(entry); published > 0 {
				ports = append(ports, published)
			}
		}
	}
	return ports, nil
}

// ValidatePort ensures the configured port is published by the compose file.
func ValidatePort(composePath string, port int) error {
	if port <= 0 {
		return fmt.Errorf("compose workloads require an explicit port in gare configuration")
	}
	ports, err := PublishedPorts(composePath)
	if err != nil {
		return err
	}
	if slices.Contains(ports, port) {
		return nil
	}
	if len(ports) == 0 {
		return fmt.Errorf("compose file %s publishes no host ports, cannot route port %d", filepath.Base(composePath), port)
	}
	return fmt.Errorf("port %d is not published by %s (published: %v)", port, filepath.Base(composePath), ports)
}

// readDocument parses a compose file and merges the ports gare publishes through its override file,
// so every reader sees the stack as the provider will run it.
func readDocument(composePath string, publishes ...Publish) (*document, error) {
	data, err := os.ReadFile(composePath)
	if err != nil {
		return nil, err
	}
	var doc document
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", composePath, err)
	}
	for _, publish := range publishes {
		svc, ok := doc.Services[publish.Service]
		if !ok {
			continue
		}
		svc.Ports = append(svc.Ports, publish.spec())
		doc.Services[publish.Service] = svc
	}
	return &doc, nil
}

func portPair(entry any) (int, int) {
	switch value := entry.(type) {
	case int:
		return value, value
	case string:
		return parsePortSpec(value)
	case map[string]any:
		return toPort(value["published"]), toPort(value["target"])
	}
	return 0, 0
}
