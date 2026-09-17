package storage

// ResolvePort returns the configured port or 0 if unset.
func (g *GareFile) ResolvePort() int {
	if g == nil {
		return 0
	}
	return g.Port
}

// ResolveContainerPort returns the container internal port or 0 if unset.
func (g *GareFile) ResolveContainerPort() int {
	if g == nil {
		return 0
	}
	return g.ContainerPort
}

// ResolveDomain returns the configured domain or empty string if unset.
func (g *GareFile) ResolveDomain() string {
	if g == nil {
		return ""
	}
	return g.Domain
}
