package storage

// ResolvePort returns the configured port or 0 if unset.
func (g *GareFile) ResolvePort() int {
	if g == nil {
		return 0
	}
	return g.Port
}

// ResolveDomain returns the configured domain or empty string if unset.
func (g *GareFile) ResolveDomain() string {
	if g == nil {
		return ""
	}
	return g.Domain
}
