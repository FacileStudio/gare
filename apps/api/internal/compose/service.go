package compose

import (
	"fmt"
	"path/filepath"
	"regexp"
	"slices"

	"github.com/FacileStudio/gare/apps/api/internal/atomicfile"
	"gopkg.in/yaml.v3"
)

// OverrideFileName is the file gare writes in an app's storage directory to publish the routed
// service's port, so the repository's own compose file runs unmodified.
const OverrideFileName = "compose.gare.yml"

var networkNameRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]*$`)

// Publish is the host port gare publishes for one service through its override file. It is bound
// to loopback because only the host's ingress is meant to reach it.
type Publish struct {
	Service       string
	HostPort      int
	ContainerPort int
}

// spec renders the publish as a compose short-syntax port mapping.
func (p Publish) spec() string {
	return fmt.Sprintf("127.0.0.1:%d:%d", p.HostPort, p.ContainerPort)
}

// ValidateService checks that the compose file declares the service gare routes to and that its
// container port is a real port, before anything is written from them.
func ValidateService(composePath, name string, containerPort int) error {
	if containerPort < 1 || containerPort > 65535 {
		return fmt.Errorf("compose service %q requires container_port between 1 and 65535 in gare.yml, got %d",
			name, containerPort)
	}
	doc, err := readDocument(composePath)
	if err != nil {
		return err
	}
	if _, ok := doc.Services[name]; ok {
		return nil
	}
	names := make([]string, 0, len(doc.Services))
	for declared := range doc.Services {
		names = append(names, declared)
	}
	slices.Sort(names)
	return fmt.Errorf("service %q is not declared in %s (services: %v)", name, filepath.Base(composePath), names)
}

// WriteOverride writes the override file that publishes the routed service's port and nothing else,
// so `podman compose -f <file> -f <override>` merges it into the repository's own definition.
func WriteOverride(path string, publish Publish) error {
	doc := map[string]any{"services": map[string]any{
		publish.Service: map[string]any{"ports": []string{publish.spec()}},
	}}
	data, err := yaml.Marshal(doc)
	if err != nil {
		return err
	}
	return atomicfile.WriteFile(path, data, 0644)
}

// ExternalNetworks returns the names of the networks a compose file declares external, sorted. The
// provider refuses to start a stack whose external network is missing, so gare creates them first.
// A name podman would refuse, or one that would split a unit line, is rejected.
func ExternalNetworks(composePath string) ([]string, error) {
	doc, err := readDocument(composePath)
	if err != nil {
		return nil, err
	}
	var names []string
	for key, declared := range doc.Networks {
		if !declared.isExternal() {
			continue
		}
		name := declared.resolvedName(key)
		if !networkNameRegex.MatchString(name) {
			return nil, fmt.Errorf("external network name %q in %s is not a valid podman network name",
				name, filepath.Base(composePath))
		}
		names = append(names, name)
	}
	slices.Sort(names)
	return slices.Compact(names), nil
}
