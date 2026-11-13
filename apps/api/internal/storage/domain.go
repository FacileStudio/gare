package storage

import (
	"encoding/json"
	"fmt"
	"regexp"
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

// AddDomain validates and stores a hostname on the app, rejecting one that conflicts with a domain
// it already has, since Caddy refuses a site address listed twice.
func (c *AppConfig) AddDomain(hostname string) error {
	if err := ValidateDomain(hostname); err != nil {
		return err
	}
	normalized := NormalizeDomain(hostname)
	if AppByDomain([]*AppConfig{c}, normalized) != nil {
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
