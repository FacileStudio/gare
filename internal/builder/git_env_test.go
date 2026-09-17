package builder

import (
	"testing"
)

func TestGitEnviron(t *testing.T) {
	env := gitEnviron()
	hasPrompt := false
	hasSSH := false
	for _, e := range env {
		if e == "GIT_TERMINAL_PROMPT=0" {
			hasPrompt = true
		}
		if e == "GIT_SSH_COMMAND=ssh -o BatchMode=yes" {
			hasSSH = true
		}
	}
	if !hasPrompt {
		t.Error("expected GIT_TERMINAL_PROMPT=0 in gitEnviron")
	}
	if !hasSSH {
		t.Error("expected GIT_SSH_COMMAND=ssh -o BatchMode=yes in gitEnviron")
	}
}

func TestGitAuthTokens(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "test-gh-token")
	if tok := githubToken(); tok != "test-gh-token" {
		t.Errorf("expected test-gh-token, got %s", tok)
	}
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv("GH_TOKEN", "test-gh-alt")
	if tok := githubToken(); tok != "test-gh-alt" {
		t.Errorf("expected test-gh-alt, got %s", tok)
	}

	t.Setenv("GITLAB_TOKEN", "test-gl-token")
	if tok := gitlabToken(); tok != "test-gl-token" {
		t.Errorf("expected test-gl-token, got %s", tok)
	}
	t.Setenv("GITLAB_TOKEN", "")
	t.Setenv("GL_TOKEN", "test-gl-alt")
	if tok := gitlabToken(); tok != "test-gl-alt" {
		t.Errorf("expected test-gl-alt, got %s", tok)
	}
}
