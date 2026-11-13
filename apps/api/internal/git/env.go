package git

import (
	"os"
	"strings"
)

func gitEnviron() []string {
	env := os.Environ()
	var filtered []string
	for _, e := range env {
		if !strings.HasPrefix(e, "GIT_TERMINAL_PROMPT=") && !strings.HasPrefix(e, "GIT_SSH_COMMAND=") {
			filtered = append(filtered, e)
		}
	}
	filtered = append(filtered, "GIT_TERMINAL_PROMPT=0", "GIT_SSH_COMMAND=ssh -o BatchMode=yes")
	return filtered
}

// tokenEnv returns the first of the named environment variables that holds a token, or empty when
// none does.
func tokenEnv(names ...string) string {
	for _, name := range names {
		if os.Getenv(name) != "" {
			return name
		}
	}
	return ""
}
