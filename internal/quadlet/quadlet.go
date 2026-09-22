package quadlet

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// GeneratorPath returns the path of the installed Podman Quadlet generator.
func GeneratorPath() (string, error) {
	candidates := generatorCandidates()
	path, err := generatorPathFrom(candidates)
	if err == nil {
		return path, nil
	}
	if found, lookErr := exec.LookPath("podman-system-generator"); lookErr == nil {
		return found, nil
	}
	return "", err
}

// CheckGenerator verifies that Podman can generate systemd units from Quadlet sources.
func CheckGenerator() error {
	_, err := GeneratorPath()
	return err
}

// Remove deletes every Quadlet source file gare owns for an application.
func Remove(name string) error {
	return retireSiblingSources(name, "")
}

// retireSiblingSources removes the application's other Quadlet source, so a workload type
// change cannot leave two sources generating the same unit name.
func retireSiblingSources(name, keep string) error {
	for _, path := range []string{KubePath(name), ContainerPath(name)} {
		if path == keep {
			continue
		}
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func generatorPathFrom(candidates []string) (string, error) {
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("podman quadlet generator not found — install podman 4.4 or newer (looked in %s)",
		strings.Join(candidates, ", "))
}

func generatorCandidates() []string {
	return []string{
		"/usr/lib/systemd/user-generators/podman-user-generator",
		"/usr/lib/systemd/system-generators/podman-system-generator",
		"/usr/libexec/podman/quadlet",
		"/usr/local/libexec/podman/quadlet",
	}
}
