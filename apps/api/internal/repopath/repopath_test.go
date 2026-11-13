package repopath

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidate(t *testing.T) {
	cases := []struct {
		path    string
		wantErr bool
	}{
		{".", false},
		{"dist", false},
		{"apps/api/Dockerfile", false},
		{"a/../b", false},
		{"foo..yml", false},
		{"", true},
		{"   ", true},
		{"with space", true},
		{"tab\tdir", true},
		{"../x", true},
		{"..", true},
		{"a/../../x", true},
		{"/etc", true},
		{"/srv/../etc", true},
	}
	for _, tc := range cases {
		err := Validate("static_dir", tc.path)
		if (err != nil) != tc.wantErr {
			t.Errorf("Validate(%q) error = %v, wantErr %v", tc.path, err, tc.wantErr)
		}
	}
}

func TestJoinRejectsASymlinkOutOfTheRepository(t *testing.T) {
	base := t.TempDir()
	repoDir := filepath.Join(base, "repo")
	outside := filepath.Join(base, "otherapp")
	for _, dir := range []string{filepath.Join(repoDir, "public"), outside} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(outside, filepath.Join(repoDir, "dist")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("public", filepath.Join(repoDir, "site")); err != nil {
		t.Fatal(err)
	}
	if _, err := Join("static_dir", repoDir, "dist"); err == nil {
		t.Error("expected a symlink out of the repository to be rejected")
	}
	for _, path := range []string{"", ".", "public", "site", "not-built-yet"} {
		got, err := Join("static_dir", repoDir, path)
		if err != nil || got != filepath.Join(repoDir, path) {
			t.Errorf("Join(%q) = %q, %v", path, got, err)
		}
	}
}
