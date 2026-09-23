package compose

import (
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/FacileStudio/gare/internal/health"
)

var healthcheckURLRegex = regexp.MustCompile(`https?://[^\s"']+`)

// HealthProbes derives HTTP probes from HTTP healthchecks declared in a compose file. Only services
// whose healthcheck reaches a published container port qualify, because a container-internal check
// cannot be reached from the host.
func HealthProbes(composePath string) ([]health.Probe, error) {
	doc, err := readDocument(composePath)
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(doc.Services))
	for name := range doc.Services {
		names = append(names, name)
	}
	slices.Sort(names)

	var probes []health.Probe
	for _, name := range names {
		if probe, ok := probeFromHealthcheck(name, doc.Services[name]); ok {
			probes = append(probes, probe)
		}
	}
	return probes, nil
}

func probeFromHealthcheck(name string, svc service) (health.Probe, bool) {
	target := healthcheckURL(healthcheckCommand(svc.Healthcheck))
	if target == "" {
		return health.Probe{}, false
	}
	parsed, err := url.Parse(target)
	if err != nil {
		return health.Probe{}, false
	}
	hostPort := servicePortMap(svc.Ports)[urlPort(parsed)]
	if hostPort <= 0 {
		return health.Probe{}, false
	}
	return health.Probe{Name: name, Port: hostPort, Path: parsed.Path}, true
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

func healthcheckCommand(check *healthcheck) string {
	if check == nil {
		return ""
	}
	switch test := check.Test.(type) {
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
