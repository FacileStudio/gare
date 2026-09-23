package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func assertWorkloadType(t *testing.T, gf *GareFile, want string) {
	t.Helper()
	workload, err := gf.ResolveWorkload()
	if err != nil {
		t.Fatalf("unexpected workload resolution error: %v", err)
	}
	if string(workload) != want {
		t.Errorf("workload type: got %q, want %q", workload, want)
	}
}

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
	assertWorkloadType(t, gf, "static")
	if gf.Containerfile != "Containerfile.dev" ||
		gf.Context != "./src" || gf.StaticDir != "dist" || gf.BuildCmd != "npm run build" {
		t.Errorf("unexpected parsed values from gare.yaml: %+v", gf)
	}

	testLoadGareYmlAndInvalid(t, tmpDir)
}

func testLoadGareYmlAndInvalid(t *testing.T, tmpDir string) {
	ymlDir := filepath.Join(tmpDir, "ymldir")
	if err := os.MkdirAll(ymlDir, 0755); err != nil {
		t.Fatal(err)
	}
	ymlContent := "type: container\ncontainerfile: Dockerfile\ncontext: .\nstatic_dir: build\nbuild_cmd: cargo build\n"
	if err := os.WriteFile(filepath.Join(ymlDir, "gare.yml"), []byte(ymlContent), 0644); err != nil {
		t.Fatal(err)
	}
	gf, err := LoadGareFile(ymlDir)
	if err != nil || gf == nil {
		t.Fatalf("failed to load gare.yml: %v", err)
	}
	assertWorkloadType(t, gf, "container")
	if gf.Containerfile != "Dockerfile" ||
		gf.Context != "." || gf.StaticDir != "build" || gf.BuildCmd != "cargo build" {
		t.Errorf("unexpected parsed values from gare.yml: %+v", gf)
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

func TestGareFileFlatFields(t *testing.T) {
	var nilGf *GareFile
	assertWorkloadType(t, nilGf, "container")
	if nilGf.ResolveComposeFile() != "" {
		t.Errorf("nil ResolveComposeFile: got %q, want empty", nilGf.ResolveComposeFile())
	}

	gf := &GareFile{
		Type: "static", Containerfile: "nested.dockerfile", Context: "nested-context",
		StaticDir: "nested-static", BuildCmd: "nested-build", Healthcheck: "/healthz",
	}
	assertWorkloadType(t, gf, "static")
	if gf.Containerfile != "nested.dockerfile" || gf.Context != "nested-context" {
		t.Errorf("unexpected containerfile/context: %+v", gf)
	}
	if gf.StaticDir != "nested-static" || gf.BuildCmd != "nested-build" || gf.Healthcheck != "/healthz" {
		t.Errorf("unexpected static dir/build cmd/healthcheck: %+v", gf)
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
