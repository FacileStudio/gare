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

func githubToken() string {
	if tok := os.Getenv("GITHUB_TOKEN"); tok != "" {
		return tok
	}
	return os.Getenv("GH_TOKEN")
}

func gitlabToken() string {
	if tok := os.Getenv("GITLAB_TOKEN"); tok != "" {
		return tok
	}
	return os.Getenv("GL_TOKEN")
}
