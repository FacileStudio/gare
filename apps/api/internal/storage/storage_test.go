package storage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAppPaths(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", "")
	baseDir := DefaultBaseDir()
	if baseDir == "" || !strings.HasSuffix(baseDir, filepath.Join(".local", "share", "gare", "apps")) {
		t.Fatalf("unexpected baseDir %q", baseDir)
	}
	if got := Open("/base/dir").AppDir("testapp"); got != "/base/dir/testapp" {
		t.Errorf("AppDir: got %q", got)
	}
	if got := Default().AppDir("testapp"); got != filepath.Join(baseDir, "testapp") {
		t.Errorf("Default AppDir: got %q", got)
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
		Domains: []string{"myapp.example.com"}, Port: 8080, Branch: "main", CreatedAt: "2026-09-15T00:00:00Z",
		AppType: "static", Containerfile: "Dockerfile", ContextDir: ".", StaticDir: "dist", BuildCmd: "make",
	}
	if err := saveConfig(appDir, cfg); err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}
	loaded, err := loadConfig(appDir)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	assertConfigEqual(t, loaded, cfg)
	info, err := os.Stat(GetConfigPath(appDir))
	if err != nil || info.Mode().Perm() != 0600 {
		t.Errorf("config.json mode: %v (err %v), want 0600", info.Mode().Perm(), err)
	}
	testInvalidConfigs(t, tmpDir)
}

func assertConfigEqual(t *testing.T, got, want *AppConfig) {
	if got.Name != want.Name || got.RepoURL != want.RepoURL || !domainsEqual(got.Domains, want.Domains) ||
		got.Port != want.Port || got.Branch != want.Branch || got.CreatedAt != want.CreatedAt ||
		got.AppType != want.AppType || got.Containerfile != want.Containerfile ||
		got.ContextDir != want.ContextDir || got.StaticDir != want.StaticDir || got.BuildCmd != want.BuildCmd {
		t.Errorf("config mismatch:\ngot:  %+v\nwant: %+v", got, want)
	}
}

func domainsEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func testInvalidConfigs(t *testing.T, tmpDir string) {
	if _, err := loadConfig(filepath.Join(tmpDir, "nonexistent")); err == nil {
		t.Error("expected error loading non-existent config, got nil")
	}
	badDir := filepath.Join(tmpDir, "bad")
	if err := os.MkdirAll(badDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(GetConfigPath(badDir), []byte("not valid json"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := loadConfig(badDir); err == nil {
		t.Error("expected unmarshal error for invalid json, got nil")
	}
}

func TestListApps(t *testing.T) {
	tmpDir := t.TempDir()
	apps, err := Open(tmpDir).List()
	if err != nil || len(apps) != 0 {
		t.Fatalf("expected 0 apps on empty dir, got %d (err: %v)", len(apps), err)
	}
	apps, err = Open(filepath.Join(tmpDir, "nonexistent")).List()
	if err != nil || len(apps) != 0 {
		t.Fatalf("expected 0 apps on nonexistent dir, got %d (err: %v)", len(apps), err)
	}

	setupListAppsFixtures(t, tmpDir)
	apps, err = Open(tmpDir).List()
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
		if err := saveConfig(appDir, &AppConfig{Name: name, Port: 8000}); err != nil {
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

func TestDelete(t *testing.T) {
	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "myapp")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatal(err)
	}
	filePath := filepath.Join(appDir, "config.json")
	if err := os.WriteFile(filePath, []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := Open(tmpDir).Delete("myapp"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	if _, err := os.Stat(appDir); !os.IsNotExist(err) {
		t.Errorf("expected appDir to be deleted")
	}
	if err := Open(tmpDir).Delete("myapp"); err != nil {
		t.Errorf("Delete on nonexistent dir error: %v", err)
	}
	if err := Open(tmpDir).Delete(""); err == nil {
		t.Errorf("Delete with an empty name must refuse rather than remove the store")
	}
}

func TestStaticConfigStorage(t *testing.T) {
	tmpDir := t.TempDir()
	appDir := filepath.Join(tmpDir, "static-app")
	cfg := &AppConfig{
		Name:      "static-app",
		Domains:   []string{"static.test"},
		AppType:   "static",
		StaticDir: "public",
		BuildCmd:  "npm run build",
	}
	if err := Open(filepath.Dir(appDir)).Save(cfg); err != nil {
		t.Fatal(err)
	}
	loaded, err := Open(filepath.Dir(appDir)).Load(filepath.Base(appDir))
	if err != nil {
		t.Fatal(err)
	}
	if loaded.AppType != "static" || loaded.StaticDir != "public" {
		t.Errorf("expected static app with public staticDir, got: %+v", loaded)
	}
}
