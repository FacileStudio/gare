package appname

import (
	"fmt"
	"regexp"
)

var validName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,62}$`)

// Validate rejects an application name that could not be a single path segment, systemd unit name
// or container name, so the CLI, the webhook server and storage all derive from one rule.
func Validate(name string) error {
	if !validName.MatchString(name) {
		return fmt.Errorf("invalid app name %q: must match ^[a-zA-Z0-9][a-zA-Z0-9_-]{0,62}$", name)
	}
	return nil
}
