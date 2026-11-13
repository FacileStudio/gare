package notify

import (
	"regexp"
	"strings"
)

// maxMessage caps an alert's message: a failed deploy's error can carry a build's output, and an
// alert is a pointer to the deploy log, not a copy of it.
const maxMessage = 2000

// secretAssignment matches a KEY=value whose key names a credential, the shape an env dump or a
// failing command line leaves in a deploy error.
var secretAssignment = regexp.MustCompile(`(?i)[A-Z0-9_]*(SECRET|TOKEN|PASSWORD|KEY)\s*=`)

// Redact drops every line of msg that assigns a secret-looking variable, then caps what is left at
// maxMessage runes, so an alert never carries a credential the deploy output happened to print.
func Redact(msg string) string {
	lines := strings.Split(msg, "\n")
	kept := lines[:0]
	for _, line := range lines {
		if !secretAssignment.MatchString(line) {
			kept = append(kept, line)
		}
	}
	runes := []rune(strings.Join(kept, "\n"))
	if len(runes) > maxMessage {
		runes = runes[:maxMessage]
	}
	return string(runes)
}
