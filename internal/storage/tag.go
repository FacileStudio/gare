package storage

import (
	"fmt"
	"regexp"
	"strings"
)

var tagRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,62}$`)

// ValidateTag validates that a tag matches allowed characters and length.
func ValidateTag(tag string) error {
	normalized := NormalizeTag(tag)
	if normalized == "" {
		return fmt.Errorf("tag must not be empty")
	}
	if !tagRegex.MatchString(normalized) {
		return fmt.Errorf("invalid tag %q: must match ^[a-zA-Z0-9][a-zA-Z0-9_-]{0,62}$", tag)
	}
	return nil
}

// NormalizeTag trims whitespace and lowercases the tag.
func NormalizeTag(tag string) string {
	return strings.ToLower(strings.TrimSpace(tag))
}

// AddTags validates and adds tags to the AppConfig, ignoring duplicates.
func (c *AppConfig) AddTags(tags ...string) error {
	if c == nil {
		return fmt.Errorf("cannot add tags to nil config")
	}
	for _, tag := range tags {
		if err := ValidateTag(tag); err != nil {
			return err
		}
	}
	for _, tag := range tags {
		normalized := NormalizeTag(tag)
		if !c.HasTag(normalized) {
			c.Tags = append(c.Tags, normalized)
		}
	}
	return nil
}

// RemoveTags removes specified tags from the AppConfig.
func (c *AppConfig) RemoveTags(tags ...string) bool {
	if c == nil {
		return false
	}
	changed := false
	for _, tag := range tags {
		normalized := NormalizeTag(tag)
		for i, existing := range c.Tags {
			if NormalizeTag(existing) == normalized {
				c.Tags = append(c.Tags[:i], c.Tags[i+1:]...)
				changed = true
				break
			}
		}
	}
	return changed
}

// HasTag reports whether the application carries the specified tag.
func (c *AppConfig) HasTag(tag string) bool {
	if c == nil {
		return false
	}
	normalized := NormalizeTag(tag)
	for _, existing := range c.Tags {
		if NormalizeTag(existing) == normalized {
			return true
		}
	}
	return false
}

// FilterAppsByTag returns all applications matching the given tag.
func FilterAppsByTag(apps []*AppConfig, tag string) []*AppConfig {
	normalized := NormalizeTag(tag)
	if normalized == "" {
		return apps
	}
	filtered := make([]*AppConfig, 0)
	for _, app := range apps {
		if app != nil && app.HasTag(normalized) {
			filtered = append(filtered, app)
		}
	}
	return filtered
}

// CollectAllTags counts applications grouped by tag.
func CollectAllTags(apps []*AppConfig) map[string]int {
	counts := make(map[string]int)
	for _, app := range apps {
		if app == nil {
			continue
		}
		seen := make(map[string]struct{}, len(app.Tags))
		for _, tag := range app.Tags {
			normalized := NormalizeTag(tag)
			if normalized != "" {
				seen[normalized] = struct{}{}
			}
		}
		for tag := range seen {
			counts[tag]++
		}
	}
	return counts
}
