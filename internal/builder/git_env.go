package builder

import (
	"fmt"
	"os"
	"os/exec"
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

func gitAuthArgs() []string {
	var args []string
	if _, err := exec.LookPath("gh"); err == nil {
		args = append(args, "-c", "credential.https://github.com.helper=", "-c", "credential.https://github.com.helper=!gh auth git-credential")
	} else if token := githubToken(); token != "" {
		h := fmt.Sprintf("!f() { echo username=x-access-token; echo password=%s; }; f", token)
		args = append(args, "-c", "credential.https://github.com.helper=", "-c", "credential.https://github.com.helper="+h)
	}

	if _, err := exec.LookPath("glab"); err == nil {
		args = append(args, "-c", "credential.https://gitlab.com.helper=", "-c", "credential.https://gitlab.com.helper=!glab auth git-credential")
	} else if token := gitlabToken(); token != "" {
		h := fmt.Sprintf("!f() { echo username=oauth2; echo password=%s; }; f", token)
		args = append(args, "-c", "credential.https://gitlab.com.helper=", "-c", "credential.https://gitlab.com.helper="+h)
	}
	return args
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
