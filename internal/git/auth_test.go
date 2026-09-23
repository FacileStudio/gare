package git

import (
	"strings"
	"testing"
)

func isolateCredentials(t *testing.T) {
	t.Helper()
	t.Setenv("PATH", t.TempDir())
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv("GH_TOKEN", "")
	t.Setenv("GITLAB_TOKEN", "")
	t.Setenv("GL_TOKEN", "")
}

func authArgs(auth Auth) string {
	return strings.Join(auth.args(), " ")
}

func TestAuthArgsWithoutCredentials(t *testing.T) {
	isolateCredentials(t)
	if got := authArgs(Auth{}); got != "" {
		t.Errorf("no CLI and no token must configure no helper, got %q", got)
	}
}

func TestAuthArgsProviderFiltering(t *testing.T) {
	isolateCredentials(t)
	t.Setenv("GITHUB_TOKEN", "gh-secret")
	t.Setenv("GITLAB_TOKEN", "gl-secret")

	both := authArgs(Auth{})
	if !strings.Contains(both, "github.com") || !strings.Contains(both, "gitlab.com") {
		t.Errorf("an unset provider must configure both hosts, got %q", both)
	}
	github := authArgs(Auth{Provider: "GITHUB"})
	if !strings.Contains(github, "github.com") || strings.Contains(github, "gitlab.com") {
		t.Errorf("github provider must only configure github credentials, got %q", github)
	}
	gitlab := authArgs(Auth{Provider: "gitlab"})
	if !strings.Contains(gitlab, "gitlab.com") || strings.Contains(gitlab, "github.com") {
		t.Errorf("gitlab provider must only configure gitlab credentials, got %q", gitlab)
	}
}

func TestAuthArgsForcedCLI(t *testing.T) {
	isolateCredentials(t)
	auth := Auth{UseGitHubCLI: true}
	got := authArgs(auth)
	if !strings.Contains(got, "!gh auth git-credential") {
		t.Errorf("--use-github-cli must select the gh helper even without it on PATH, got %q", got)
	}
	if strings.Contains(got, "glab") {
		t.Errorf("--use-github-cli must not force the gitlab CLI, got %q", got)
	}
}

func TestAuthArgsCredentialHelperOverride(t *testing.T) {
	isolateCredentials(t)
	got := authArgs(Auth{Provider: "github", CredentialHelper: "!store --file=/tmp/creds"})
	if !strings.Contains(got, "credential.https://github.com.helper=!store --file=/tmp/creds") {
		t.Errorf("--credential-helper must be used verbatim, got %q", got)
	}
}

func TestAuthArgsTokenFallback(t *testing.T) {
	isolateCredentials(t)
	t.Setenv("GITHUB_TOKEN", "secret")
	got := authArgs(Auth{Provider: "github"})
	if !strings.Contains(got, "password=secret") || !strings.Contains(got, "username=x-access-token") {
		t.Errorf("a token must configure a credential helper, got %q", got)
	}
}

func TestAuthValidate(t *testing.T) {
	if err := (Auth{Provider: "githubb"}).Validate(); err == nil {
		t.Error("expected an error for an unknown provider")
	}
	for _, provider := range []string{"", "github", "gitlab", " GitLab "} {
		if err := (Auth{Provider: provider}).Validate(); err != nil {
			t.Errorf("provider %q must validate, got %v", provider, err)
		}
	}
}
