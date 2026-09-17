package storage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const expectedManifest = `apiVersion: v1
kind: Pod
metadata:
  name: webapp
  labels:
    app: webapp
spec:
  containers:
  - name: webapp
    image: localhost/webapp:latest
    ports:
    - containerPort: 3011
      hostPort: 8080
`

func TestAppPaths(t *testing.T) {
	baseDir := DefaultBaseDir()
	if baseDir == "" || !strings.HasSuffix(baseDir, filepath.Join(".local", "share", "gare", "apps")) {
		t.Fatalf("unexpected baseDir %q", baseDir)
	}
	if got := GetAppDir("/base/dir", "testapp"); got != "/base/dir/testapp" {
		t.Errorf("GetAppDir: got %q", got)
	}
	if got := GetAppDir("", "testapp"); got != filepath.Join(baseDir, "testapp") {
		t.Errorf("GetAppDir default: got %q", got)
	}
	appDir := "/base/testapp"
	if got := GetRepoDir(appDir); got != filepath.Join(appDir, "repo") {
		t.Errorf("GetRepoDir: got %q", got)
	}
	if got := GetManifestPath(appDir); got != filepath.Join(appDir, "manifest.yaml") {
		t.Errorf("GetManifestPath: got %q", got)
	}
	if got := GetConfigPath(appDir); got != filepath.Join(appDir, "config.json") {
		t.Errorf("GetConfigPath: got %q", got)
	}
}

func TestSaveAndLoadConfig(t *testing.T) {
	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "myapp")
	cfg := &AppConfig{
		Name: "myapp", RepoURL: "https://github.com/example/repo.git",
		Domain: "myapp.example.com", Port: 8080, Branch: "main", CreatedAt: "2026-09-15T00:00:00Z",
		AppType: "static", Containerfile: "Dockerfile", ContextDir: ".", StaticDir: "dist", BuildCmd: "make",
	}
	if err := SaveConfig(appDir, cfg); err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}
	loaded, err := LoadConfig(appDir)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	assertConfigEqual(t, loaded, cfg)

	loadedDirect, err := LoadConfig(filepath.Join(appDir, "config.json"))
	if err != nil {
		t.Fatalf("LoadConfig directly failed: %v", err)
	}
	assertConfigEqual(t, loadedDirect, cfg)
	testInvalidConfigs(t, tmpDir)
}

func assertConfigEqual(t *testing.T, got, want *AppConfig) {
	if got.Name != want.Name || got.RepoURL != want.RepoURL || got.Domain != want.Domain ||
		got.Port != want.Port || got.Branch != want.Branch || got.CreatedAt != want.CreatedAt ||
		got.AppType != want.AppType || got.Containerfile != want.Containerfile ||
		got.ContextDir != want.ContextDir || got.StaticDir != want.StaticDir || got.BuildCmd != want.BuildCmd {
		t.Errorf("config mismatch:\ngot:  %+v\nwant: %+v", got, want)
	}
}

func testInvalidConfigs(t *testing.T, tmpDir string) {
	if _, err := LoadConfig(filepath.Join(tmpDir, "nonexistent")); err == nil {
		t.Error("expected error loading non-existent config, got nil")
	}
	badPath := filepath.Join(tmpDir, "bad.json")
	if err := os.WriteFile(badPath, []byte("not valid json"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadConfig(badPath); err == nil {
		t.Error("expected unmarshal error for invalid json, got nil")
	}
}

func TestGenerateDefaultManifest(t *testing.T) {
	tmpDir := t.TempDir()
	manifestFile := filepath.Join(tmpDir, "manifest.yaml")
	if err := GenerateDefaultManifest("webapp", 3011, 8080, manifestFile); err != nil {
		t.Fatalf("GenerateDefaultManifest failed: %v", err)
	}
	content, err := os.ReadFile(manifestFile)
	if err != nil || string(content) != expectedManifest {
		t.Fatalf("manifest content mismatch: %v", err)
	}

	appDir := filepath.Join(tmpDir, "appwithdir")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := GenerateDefaultManifest("webapp2", 3011, 9000, appDir); err != nil {
		t.Fatalf("GenerateDefaultManifest with dir failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(appDir, "manifest.yaml")); err != nil {
		t.Fatalf("expected manifest.yaml to be created: %v", err)
	}
}

func TestListApps(t *testing.T) {
	tmpDir := t.TempDir()
	apps, err := ListApps(tmpDir)
	if err != nil || len(apps) != 0 {
		t.Fatalf("expected 0 apps on empty dir, got %d (err: %v)", len(apps), err)
	}
	apps, err = ListApps(filepath.Join(tmpDir, "nonexistent"))
	if err != nil || len(apps) != 0 {
		t.Fatalf("expected 0 apps on nonexistent dir, got %d (err: %v)", len(apps), err)
	}

	setupListAppsFixtures(t, tmpDir)
	apps, err = ListApps(tmpDir)
	if err != nil || len(apps) != 3 {
		t.Fatalf("expected 3 apps, got %d (err: %v)", len(apps), err)
	}
	if apps[0].Name != "alpha" || apps[1].Name != "beta" || apps[2].Name != "gamma" {
		t.Errorf("expected [alpha, beta, gamma], got [%s, %s, %s]",
			apps[0].Name, apps[1].Name, apps[2].Name)
	}
}

func setupListAppsFixtures(t *testing.T, tmpDir string) {
	for _, name := range []string{"beta", "alpha", "gamma"} {
		appDir := filepath.Join(tmpDir, name)
		if err := SaveConfig(appDir, &AppConfig{Name: name, Port: 8000}); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "regular_file.txt"), []byte("hi"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, "empty_dir"), 0755); err != nil {
		t.Fatal(err)
	}
}

func TestDeleteAppStorage(t *testing.T) {
	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "myapp")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatal(err)
	}
	filePath := filepath.Join(appDir, "config.json")
	if err := os.WriteFile(filePath, []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := DeleteAppStorage(appDir); err != nil {
		t.Fatalf("DeleteAppStorage failed: %v", err)
	}
	if _, err := os.Stat(appDir); !os.IsNotExist(err) {
		t.Errorf("expected appDir to be deleted")
	}
	if err := DeleteAppStorage(appDir); err != nil {
		t.Errorf("DeleteAppStorage on nonexistent dir error: %v", err)
	}
}

func TestValidateAppName(t *testing.T) {
	cases := []struct {
		name    string
		wantErr bool
	}{
		{"a", false},
		{"my-app", false},
		{"my_app-123", false},
		{strings.Repeat("a", 63), false},
		{"", true},
		{"../etc", true},
		{"../../bad", true},
		{"-leading-dash", true},
		{"_leading-under", true},
		{"has spaces", true},
		{"invalid@char", true},
		{"invalid.dot", true},
		{strings.Repeat("a", 64), true},
	}
	for _, tc := range cases {
		err := ValidateAppName(tc.name)
		if (err != nil) != tc.wantErr {
			t.Errorf("ValidateAppName(%q) error = %v, wantErr %v", tc.name, err, tc.wantErr)
		}
	}
}
