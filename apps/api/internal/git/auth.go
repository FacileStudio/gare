package git

import (
	"fmt"
	"os/exec"
	"strings"
)

// Auth describes how gare authenticates against the git remotes it fetches from.
type Auth struct {
	Provider         string
	UseGitHubCLI     bool
	UseGitLabCLI     bool
	CredentialHelper string
}

// Validate rejects a provider gare has no credentials for.
func (a Auth) Validate() error {
	switch strings.ToLower(strings.TrimSpace(a.Provider)) {
	case "", "github", "gitlab":
		return nil
	}
	return fmt.Errorf("unknown git provider %q: want github or gitlab", a.Provider)
}

// credential describes how to authenticate against one forge. tokenEnv names the environment
// variable holding its token rather than the token itself, so the token reaches git's credential
// helper through the child's environment and never through argv, which every user can read.
type credential struct {
	host     string
	cli      string
	forced   bool
	tokenEnv string
	user     string
}

func (a Auth) args() []string {
	var args []string
	for _, cred := range a.credentials() {
		if !a.appliesTo(cred.host) {
			continue
		}
		if helper := a.helperFor(cred); helper != "" {
			args = append(args,
				"-c", "credential.https://"+cred.host+".helper=",
				"-c", "credential.https://"+cred.host+".helper="+helper)
		}
	}
	return args
}

func (a Auth) credentials() []credential {
	return []credential{
		{host: "github.com", cli: "gh", forced: a.UseGitHubCLI, tokenEnv: tokenEnv("GITHUB_TOKEN", "GH_TOKEN"),
			user: "x-access-token"},
		{host: "gitlab.com", cli: "glab", forced: a.UseGitLabCLI, tokenEnv: tokenEnv("GITLAB_TOKEN", "GL_TOKEN"),
			user: "oauth2"},
	}
}

func (a Auth) appliesTo(host string) bool {
	switch strings.ToLower(strings.TrimSpace(a.Provider)) {
	case "":
		return true
	case "github":
		return host == "github.com"
	case "gitlab":
		return host == "gitlab.com"
	}
	return false
}

func (a Auth) helperFor(cred credential) string {
	if a.CredentialHelper != "" {
		return a.CredentialHelper
	}
	if cred.forced || hasBinary(cred.cli) {
		return "!" + cred.cli + " auth git-credential"
	}
	if cred.tokenEnv != "" {
		return fmt.Sprintf(`!f() { echo username=%s; echo "password=$%s"; }; f`, cred.user, cred.tokenEnv)
	}
	return ""
}

func hasBinary(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}
