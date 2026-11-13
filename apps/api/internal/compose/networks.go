package compose

import "strings"

// network is one entry of a compose file's top-level networks section, reduced to what tells an
// external network and its real name apart.
type network struct {
	External any    `yaml:"external"`
	Name     string `yaml:"name"`
}

// isExternal reports whether the network is declared external, either as `external: true` or in the
// legacy `external: {name: ...}` form.
func (n *network) isExternal() bool {
	if n == nil {
		return false
	}
	switch value := n.External.(type) {
	case bool:
		return value
	case map[string]any:
		return true
	}
	return false
}

// resolvedName returns the name the network has in podman: its `name`, the legacy `external.name`,
// or else the key it is declared under, since compose never prefixes an external network.
func (n *network) resolvedName(key string) string {
	if name := strings.TrimSpace(n.Name); name != "" {
		return name
	}
	if legacy, ok := n.External.(map[string]any); ok {
		if name, ok := legacy["name"].(string); ok && strings.TrimSpace(name) != "" {
			return strings.TrimSpace(name)
		}
	}
	return key
}
