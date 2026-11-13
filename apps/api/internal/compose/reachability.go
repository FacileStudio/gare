package compose

import (
	"strings"
)

// LoopbackOnly reports whether every host port a compose file publishes is bound to the loopback
// interface. Such a workload is reachable from the host but from no other machine, so it depends on
// ingress to be served externally. A file that publishes nothing reports false.
func LoopbackOnly(composePath string) (bool, error) {
	doc, err := readDocument(composePath)
	if err != nil {
		return false, err
	}
	published := false
	for _, svc := range doc.Services {
		for _, entry := range svc.Ports {
			port, _ := portPair(entry)
			if port <= 0 {
				continue
			}
			published = true
			if !isLoopbackHost(portEntryHost(entry)) {
				return false, nil
			}
		}
	}
	return published, nil
}

// portEntryHost returns the host interface a single port mapping binds to, empty for a mapping that
// names none and so binds every interface.
func portEntryHost(entry any) string {
	switch value := entry.(type) {
	case string:
		return hostFromSpec(value)
	case map[string]any:
		host, _ := value["host_ip"].(string)
		return strings.TrimSpace(host)
	}
	return ""
}

func hostFromSpec(spec string) string {
	trimmed := strings.ToLower(strings.TrimSpace(spec))
	if strings.HasPrefix(trimmed, "[") {
		if end := strings.Index(trimmed, "]"); end > 0 {
			return trimmed[1:end]
		}
	}
	parts := strings.Split(trimmed, ":")
	if len(parts) < 3 {
		return ""
	}
	return parts[0]
}

func isLoopbackHost(host string) bool {
	switch strings.ToLower(host) {
	case "127.0.0.1", "localhost", "::1":
		return true
	}
	return false
}
