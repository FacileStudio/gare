package storage

import (
	"fmt"
	"net/url"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

var healthcheckURLRegex = regexp.MustCompile(`https?://[^\s"']+`)

// ComposeHealthProbes derives HTTP probes from HTTP healthchecks declared in a compose file.
func ComposeHealthProbes(composePath string) ([]HealthProbe, error) {
	data, err := os.ReadFile(composePath)
	if err != nil {
		return nil, err
	}
	var doc composeDocument
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", composePath, err)
	}

	names := make([]string, 0, len(doc.Services))
	for name := range doc.Services {
		names = append(names, name)
	}
	slices.Sort(names)

	var probes []HealthProbe
	for _, name := range names {
		if probe, ok := probeFromHealthcheck(name, doc.Services[name]); ok {
			probes = append(probes, probe)
		}
	}
	return probes, nil
}

func probeFromHealthcheck(name string, service composeService) (HealthProbe, bool) {
	target := healthcheckURL(healthcheckCommand(service.Healthcheck))
	if target == "" {
		return HealthProbe{}, false
	}
	parsed, err := url.Parse(target)
	if err != nil {
		return HealthProbe{}, false
	}
	hostPort := servicePortMap(service.Ports)[urlPort(parsed)]
	if hostPort <= 0 {
		return HealthProbe{}, false
	}
	return HealthProbe{Name: name, Port: hostPort, Path: parsed.Path}, true
}

func servicePortMap(ports []any) map[int]int {
	mapping := make(map[int]int, len(ports))
	for _, entry := range ports {
		published, target := portPair(entry)
		if published > 0 && target > 0 {
			mapping[target] = published
		}
	}
	return mapping
}

func healthcheckCommand(healthcheck *composeHealthcheck) string {
	if healthcheck == nil {
		return ""
	}
	switch test := healthcheck.Test.(type) {
	case string:
		return test
	case []any:
		parts := make([]string, 0, len(test))
		for _, item := range test {
			if text, ok := item.(string); ok {
				parts = append(parts, text)
			}
		}
		return strings.Join(parts, " ")
	}
	return ""
}

func healthcheckURL(command string) string {
	return healthcheckURLRegex.FindString(command)
}

func urlPort(parsed *url.URL) int {
	if parsed.Port() != "" {
		port, err := strconv.Atoi(parsed.Port())
		if err != nil {
			return 0
		}
		return port
	}
	if parsed.Scheme == "https" {
		return 443
	}
	return 80
}
