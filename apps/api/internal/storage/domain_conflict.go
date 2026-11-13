package storage

import (
	"strconv"
	"strings"
)

// DomainHostPort splits a domain into the hostname an ingress serves it under and the explicit port
// it carries, reporting no port when it names none, so a probe addresses the name the ingress serves
// rather than the socket it serves it on.
func DomainHostPort(hostname string) (string, int) {
	idx := strings.LastIndex(hostname, ":")
	if idx < 0 {
		return hostname, 0
	}
	port, err := strconv.Atoi(hostname[idx+1:])
	if err != nil || port <= 0 || port > 65535 {
		return hostname, 0
	}
	return hostname[:idx], port
}

// DomainsConflict reports whether two domains would claim the same Caddy site. Hosts compare
// without their port, case-insensitively and trimmed. Caddy refuses one address declared twice and
// reads a bare host as port 443, so `example.com` and `example.com:443` are one site. A bare host
// also takes port 80 for its HTTPS redirect, so the same host is shared only on two different
// explicit ports that are neither 80 nor 443: those are distinct addresses on distinct listeners.
func DomainsConflict(a, b string) bool {
	hostA, portA := DomainHostPort(NormalizeDomain(a))
	hostB, portB := DomainHostPort(NormalizeDomain(b))
	if hostA != hostB {
		return false
	}
	if isStandardPort(portA) || isStandardPort(portB) {
		return true
	}
	return portA == portB
}

// isStandardPort reports whether a domain's port is one a bare host already serves on: none named,
// 80 or 443.
func isStandardPort(port int) bool {
	return port == 0 || port == 80 || port == 443
}

// AppByDomain scans all apps for one whose domains conflict with the given hostname.
func AppByDomain(apps []*AppConfig, hostname string) *AppConfig {
	for _, app := range apps {
		for _, owned := range app.Domains {
			if DomainsConflict(owned, hostname) {
				return app
			}
		}
	}
	return nil
}
