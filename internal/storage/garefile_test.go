package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadGareFile(t *testing.T) {
	tmpDir := t.TempDir()
	gf, err := LoadGareFile(tmpDir)
	if err != nil || gf != nil {
		t.Fatalf("expected nil, nil for missing file, got %v, %v", gf, err)
	}

	yamlDir := filepath.Join(tmpDir, "yamldir")
	if err := os.MkdirAll(yamlDir, 0755); err != nil {
		t.Fatal(err)
	}
	yamlContent := "type: static\ncontainerfile: Containerfile.dev\ncontext: ./src\nstatic_dir: dist\nbuild_cmd: npm run build\n"
	if err := os.WriteFile(filepath.Join(yamlDir, "gare.yaml"), []byte(yamlContent), 0644); err != nil {
		t.Fatal(err)
	}
	gf, err = LoadGareFile(yamlDir)
	if err != nil || gf == nil {
		t.Fatalf("failed to load gare.yaml: %v", err)
	}
	if gf.ResolveType() != "static" || gf.ResolveContainerfile() != "Containerfile.dev" ||
		gf.ResolveContext() != "./src" || gf.ResolveStaticDir() != "dist" || gf.ResolveBuildCmd() != "npm run build" {
		t.Errorf("unexpected resolved values from gare.yaml: %+v", gf)
	}

	testLoadGareYmlAndInvalid(t, tmpDir)
}

func testLoadGareYmlAndInvalid(t *testing.T, tmpDir string) {
	ymlDir := filepath.Join(tmpDir, "ymldir")
	if err := os.MkdirAll(ymlDir, 0755); err != nil {
		t.Fatal(err)
	}
	ymlContent := "type: container\nstatic:\n  dir: build\nbuild:\n  command: cargo build\n  containerfile: Dockerfile\n  context: .\n"
	if err := os.WriteFile(filepath.Join(ymlDir, "gare.yml"), []byte(ymlContent), 0644); err != nil {
		t.Fatal(err)
	}
	gf, err := LoadGareFile(ymlDir)
	if err != nil || gf == nil {
		t.Fatalf("failed to load gare.yml: %v", err)
	}
	if gf.ResolveType() != "container" || gf.ResolveContainerfile() != "Dockerfile" ||
		gf.ResolveContext() != "." || gf.ResolveStaticDir() != "build" || gf.ResolveBuildCmd() != "cargo build" {
		t.Errorf("unexpected resolved values from gare.yml: %+v", gf)
	}

	badDir := filepath.Join(tmpDir, "baddir")
	if err := os.MkdirAll(badDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(badDir, "gare.yaml"), []byte(":\n: invalid"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadGareFile(badDir); err == nil {
		t.Error("expected error for invalid yaml, got nil")
	}
}

func TestGareFileResolveMethods(t *testing.T) {
	var nilGf *GareFile
	if got := nilGf.ResolveType(); got != "container" {
		t.Errorf("nil ResolveType: got %q, want container", got)
	}
	if got := nilGf.ResolveContainerfile(); got != "" {
		t.Errorf("nil ResolveContainerfile: got %q, want empty", got)
	}
	if got := nilGf.ResolveContext(); got != "" {
		t.Errorf("nil ResolveContext: got %q, want empty", got)
	}
	if got := nilGf.ResolveStaticDir(); got != "" {
		t.Errorf("nil ResolveStaticDir: got %q, want empty", got)
	}
	if got := nilGf.ResolveBuildCmd(); got != "" {
		t.Errorf("nil ResolveBuildCmd: got %q, want empty", got)
	}
	if got := nilGf.ResolveHealthcheck(); got != "" {
		t.Errorf("nil ResolveHealthcheck: got %q, want empty", got)
	}

	testGareFileOverrides(t)
}

func testGareFileOverrides(t *testing.T) {
	gf := &GareFile{
		Type: "static", Containerfile: "flat.dockerfile", Context: "flat-context",
		StaticDir: "flat-static", BuildCmd: "flat-build", Healthcheck: "/healthz",
		Static: &StaticSection{Dir: "nested-static"},
		Build:  &BuildSection{Command: "nested-build", Containerfile: "nested.dockerfile", Context: "nested-context"},
	}
	if gf.ResolveType() != "static" || gf.ResolveContainerfile() != "nested.dockerfile" {
		t.Errorf("unexpected type/containerfile: %+v", gf)
	}
	if gf.ResolveContext() != "nested-context" || gf.ResolveStaticDir() != "nested-static" {
		t.Errorf("unexpected context/static dir: %+v", gf)
	}
	if gf.ResolveBuildCmd() != "nested-build" || gf.ResolveHealthcheck() != "/healthz" {
		t.Errorf("unexpected build cmd/healthcheck: %+v", gf)
	}
}

func TestAppConfigIsStatic(t *testing.T) {
	var nilCfg *AppConfig
	if nilCfg.IsStatic() {
		t.Errorf("nil config IsStatic: got true, want false")
	}
	cases := []struct {
		appType string
		want    bool
	}{
		{"static", true},
		{"STATIC", true},
		{"container", false},
		{"", false},
		{"other", false},
	}
	for _, tc := range cases {
		cfg := &AppConfig{AppType: tc.appType}
		if got := cfg.IsStatic(); got != tc.want {
			t.Errorf("IsStatic() for %q: got %v, want %v", tc.appType, got, tc.want)
		}
	}
}
