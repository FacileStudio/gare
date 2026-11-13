package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FacileStudio/gare/apps/api/internal/storage"
)

func TestDeriveAppName(t *testing.T) {
	cases := []struct {
		url      string
		expected string
	}{
		{"https://github.com/saravenpi/test-gare", "test-gare"},
		{"https://github.com/saravenpi/test-gare.git", "test-gare"},
		{"git@github.com:saravenpi/my-service.git", "my-service"},
		{"https://gitlab.com/group/subgroup/api-v2/", "api-v2"},
		{"custom-name", "custom-name"},
		{"", ""},
	}
	for _, tc := range cases {
		got := deriveAppName(tc.url)
		if got != tc.expected {
			t.Errorf("deriveAppName(%q) = %q, want %q", tc.url, got, tc.expected)
		}
	}
}

func TestAppCreateCmdFlagParsingNoPanic(t *testing.T) {
	root := newRootCmd("0.1.0")
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"app", "create", "--help"})
	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error running help: %v", err)
	}
}

func TestAppCreateCmdPortFlagShorthand(t *testing.T) {
	root := newRootCmd("0.1.0")
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"app", "create", "testapp", "--repo", "https://github.com/test/repo", "-p", "8080", "--help"})
	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error executing create with -p shorthand: %v", err)
	}
}

func TestStaticDeployRejectsASymlinkOutOfTheRepository(t *testing.T) {
	appDir := t.TempDir()
	repoDir := storage.GetRepoDir(appDir)
	if err := os.MkdirAll(repoDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(t.TempDir(), filepath.Join(repoDir, "dist")); err != nil {
		t.Fatal(err)
	}
	cfg := &storage.AppConfig{Name: "app", AppType: "static", StaticDir: "dist", Port: 8080}
	err := prepareStaticDeploy(t.Context(), "app", appDir, repoDir, cfg)
	if err == nil || !strings.Contains(err.Error(), "outside the repository") {
		t.Fatalf("expected a static_dir symlinked outside the repository to be rejected, got %v", err)
	}
}
