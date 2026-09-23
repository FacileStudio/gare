package storage

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var domainRegex = regexp.MustCompile(`^(\*\.)?([a-zA-Z0-9]([a-zA-Z0-9-_]{0,61}[a-zA-Z0-9])?\.)*` +
	`[a-zA-Z0-9]([a-zA-Z0-9-_]{0,61}[a-zA-Z0-9])?(:[0-9]{1,5})?$`)

type legacyConfig struct {
	Domain string `json:"domain,omitempty"`
}

// ValidateDomain validates a hostname or wildcard domain name.
func ValidateDomain(hostname string) error {
	if hostname == "" {
		return fmt.Errorf("domain must not be empty")
	}
	if strings.ContainsAny(hostname, " \t\r\n{}#;\"'\\/`$") {
		return fmt.Errorf("invalid domain %q: contains disallowed characters", hostname)
	}
	if !domainRegex.MatchString(hostname) {
		return fmt.Errorf("invalid domain %q: must be a valid domain or hostname", hostname)
	}
	return nil
}

// NormalizeDomain trims whitespace and lowercases the hostname.
func NormalizeDomain(hostname string) string {
	return strings.ToLower(strings.TrimSpace(hostname))
}

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

// AppByDomain scans all apps for one owning the given hostname.
func AppByDomain(apps []*AppConfig, hostname string) *AppConfig {
	normalized := NormalizeDomain(hostname)
	for _, app := range apps {
		if app.HasDomain(normalized) {
			return app
		}
	}
	return nil
}

// AddDomain validates and stores a hostname on the app, rejecting duplicates.
func (c *AppConfig) AddDomain(hostname string) error {
	if err := ValidateDomain(hostname); err != nil {
		return err
	}
	normalized := NormalizeDomain(hostname)
	if c.HasDomain(normalized) {
		return fmt.Errorf("app %q already owns domain %q", c.Name, normalized)
	}
	c.Domains = append(c.Domains, normalized)
	return nil
}

// RemoveDomain removes a hostname from the app, erroring if not found.
func (c *AppConfig) RemoveDomain(hostname string) error {
	normalized := NormalizeDomain(hostname)
	for i, d := range c.Domains {
		if NormalizeDomain(d) == normalized {
			c.Domains = append(c.Domains[:i], c.Domains[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("app %q does not own domain %q", c.Name, normalized)
}

// HasDomain reports whether the app owns the given hostname.
func (c *AppConfig) HasDomain(hostname string) bool {
	normalized := NormalizeDomain(hostname)
	for _, d := range c.Domains {
		if NormalizeDomain(d) == normalized {
			return true
		}
	}
	return false
}

// migrateDomain populates Domains from a legacy "domain" field if present.
func migrateDomain(cfg *AppConfig, data []byte) {
	if len(cfg.Domains) > 0 {
		return
	}
	var legacy legacyConfig
	if err := json.Unmarshal(data, &legacy); err != nil {
		return
	}
	if legacy.Domain != "" {
		cfg.Domains = []string{legacy.Domain}
	}
}
