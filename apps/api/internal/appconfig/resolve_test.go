package appconfig

import (
	"reflect"
	"testing"

	"github.com/FacileStudio/gare/apps/api/internal/health"
	"github.com/FacileStudio/gare/apps/api/internal/storage"
)

// keyCase reads one key off a resolved configuration and names the value each layer sets for it,
// in the order flag, gare.yml, stored, default.
type keyCase struct {
	key  string
	get  func(*storage.AppConfig) any
	want [4]any
}

func allFlags() Flags {
	return Flags{Type: "container", ComposeFile: "flag.yml", Containerfile: "Flag.Containerfile",
		Context: "flagctx", StaticDir: "flagdir", BuildCmd: "flag build", Healthcheck: "/flag",
		Port: 9001, ContainerPort: 3001, Tags: []string{"Flag"}}
}

func allFile() *storage.GareFile {
	return &storage.GareFile{Type: "static", ComposeFile: "file.yml", Service: "filesvc", Containerfile: "File.Containerfile",
		Context: "filectx", StaticDir: "filedir", BuildCmd: "file build", Healthcheck: "/file",
		Port: 9002, ContainerPort: 3002, Tags: []string{"file"},
		Healthchecks: []health.Probe{{Name: "file", Port: 9002, Path: "/file"}}}
}

func allStored() *storage.AppConfig {
	return &storage.AppConfig{Name: "app", AppType: "compose",
		ComposeRoute:  storage.ComposeRoute{ComposeFile: "stored.yml", Service: "storedsvc"},
		Containerfile: "Stored.Containerfile", ContextDir: "storedctx", StaticDir: "storeddir",
		BuildCmd: "stored build", Port: 9003, ContainerPort: 3003, Tags: []string{"stored"},
		Health: &storage.HealthSection{Path: "/stored", Probes: []health.Probe{{Name: "stored", Port: 9003}}}}
}

func keyCases() []keyCase {
	probes := func(c *storage.AppConfig) any {
		if c.Health == nil {
			return ""
		}
		return c.Health.Probes[0].Name
	}
	return []keyCase{
		{"type", func(c *storage.AppConfig) any { return c.AppType }, [4]any{"container", "static", "compose", "container"}},
		{"compose_file", func(c *storage.AppConfig) any { return c.ComposeFile }, [4]any{"flag.yml", "file.yml", "stored.yml", ""}},
		{"service", func(c *storage.AppConfig) any { return c.Service }, [4]any{"filesvc", "filesvc", "storedsvc", ""}},
		{"containerfile", func(c *storage.AppConfig) any { return c.Containerfile }, [4]any{"Flag.Containerfile", "File.Containerfile", "Stored.Containerfile", ""}},
		{"context", func(c *storage.AppConfig) any { return c.ContextDir }, [4]any{"flagctx", "filectx", "storedctx", ""}},
		{"static_dir", func(c *storage.AppConfig) any { return c.StaticDir }, [4]any{"flagdir", "filedir", "storeddir", ""}},
		{"build_cmd", func(c *storage.AppConfig) any { return c.BuildCmd }, [4]any{"flag build", "file build", "stored build", ""}},
		{"healthcheck", func(c *storage.AppConfig) any { return c.HealthPath() }, [4]any{"/flag", "/file", "/stored", ""}},
		{"healthchecks", probes, [4]any{"file", "file", "stored", ""}},
		{"port", func(c *storage.AppConfig) any { return c.Port }, [4]any{9001, 9002, 9003, 0}},
		{"container_port", func(c *storage.AppConfig) any { return c.ContainerPort }, [4]any{3001, 3002, 3003, 0}},
		{"tags", func(c *storage.AppConfig) any { return c.Tags }, [4]any{[]string{"flag"}, []string{"file"}, []string{"stored"}, []string(nil)}},
	}
}

func TestResolveAppliesOnePrecedenceToEveryKey(t *testing.T) {
	layers := []struct {
		label  string
		flags  Flags
		file   *storage.GareFile
		stored *storage.AppConfig
	}{
		{"flag beats gare.yml and stored", allFlags(), allFile(), allStored()},
		{"gare.yml beats stored", Flags{}, allFile(), allStored()},
		{"stored beats the default", Flags{}, &storage.GareFile{}, allStored()},
		{"default when nothing is set", Flags{}, nil, nil},
	}
	for position, layer := range layers {
		cfg, err := Resolve(layer.flags, layer.file, layer.stored)
		if err != nil {
			t.Fatalf("%s: %v", layer.label, err)
		}
		for _, kc := range keyCases() {
			if got := kc.get(cfg); !reflect.DeepEqual(got, kc.want[position]) {
				t.Errorf("%s: %s = %#v, want %#v", layer.label, kc.key, got, kc.want[position])
			}
		}
	}
}

func TestResolveTakesAGareFileChangeSinceCreate(t *testing.T) {
	stored := &storage.AppConfig{Name: "app", AppType: "static", StaticDir: "dist", BuildCmd: "make",
		Containerfile: "Containerfile"}
	file := &storage.GareFile{StaticDir: "build", BuildCmd: "make prod", Containerfile: "Dockerfile"}
	cfg, err := Resolve(Flags{}, file, stored)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.BuildCmd != "make prod" || cfg.StaticDir != "build" || cfg.Containerfile != "Dockerfile" {
		t.Errorf("gare.yml changes since create must win over stored values, got %+v", cfg)
	}
	if stored.BuildCmd != "make" || stored.StaticDir != "dist" {
		t.Errorf("Resolve must not change the stored configuration, got %+v", stored)
	}
}

func TestResolveIgnoresAFlagLeftAtItsDefault(t *testing.T) {
	file := &storage.GareFile{Port: 8080, Tags: []string{"web"}, BuildCmd: "make"}
	left, err := Resolve(Flags{}, file, nil)
	if err != nil {
		t.Fatal(err)
	}
	if left.Port != 8080 || !reflect.DeepEqual(left.Tags, []string{"web"}) || left.BuildCmd != "make" {
		t.Errorf("flags left at their default must not outrank gare.yml, got %+v", left)
	}
	passed, err := Resolve(Flags{Port: 9000, Tags: []string{"api"}, BuildCmd: "make prod"}, file, nil)
	if err != nil {
		t.Fatal(err)
	}
	if passed.Port != 9000 || !reflect.DeepEqual(passed.Tags, []string{"api"}) || passed.BuildCmd != "make prod" {
		t.Errorf("passed flags must outrank gare.yml, got %+v", passed)
	}
}

func TestResolveDefaults(t *testing.T) {
	cfg, err := Resolve(Flags{StaticDir: "dist"}, nil, nil)
	if err != nil || cfg.AppType != "static" {
		t.Fatalf("a static directory alone must infer a static workload, got %+v, %v", cfg, err)
	}
	cfg, err = Resolve(Flags{Type: "Static"}, nil, nil)
	if err != nil || cfg.AppType != "static" || cfg.StaticDir != "." {
		t.Fatalf("a static workload defaults to the repository root, got %+v, %v", cfg, err)
	}
	cfg, err = Resolve(Flags{}, &storage.GareFile{Tags: []string{}}, &storage.AppConfig{Tags: []string{"old"}})
	if err != nil || cfg.Tags != nil {
		t.Fatalf("an empty tag list in gare.yml clears the stored tags, got %+v, %v", cfg, err)
	}
}

func TestResolveRejectsAnInvalidConfiguration(t *testing.T) {
	cases := map[string]struct {
		flags Flags
		file  *storage.GareFile
	}{
		"gare.yml type":      {file: &storage.GareFile{Type: "kubernetes"}},
		"flag type":          {flags: Flags{Type: "kubernetes"}},
		"static_dir outside": {file: &storage.GareFile{StaticDir: "../x"}},
		"containerfile":      {file: &storage.GareFile{Containerfile: "/etc/passwd"}},
		"probe port":         {file: &storage.GareFile{Healthchecks: []health.Probe{{Port: 70000}}}},
		"tag":                {file: &storage.GareFile{Tags: []string{"bad tag!"}}},
	}
	for label, tc := range cases {
		if _, err := Resolve(tc.flags, tc.file, nil); err == nil {
			t.Errorf("%s: expected an error", label)
		}
	}
}
