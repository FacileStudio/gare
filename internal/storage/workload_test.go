package storage

import "testing"

func TestResolveWorkload(t *testing.T) {
	var nilGf *GareFile
	workload, err := nilGf.ResolveWorkload()
	if err != nil || workload != WorkloadContainer {
		t.Errorf("nil ResolveWorkload: got %q, err %v", workload, err)
	}

	cases := []struct {
		typ     string
		want    WorkloadType
		wantErr bool
	}{
		{"", WorkloadContainer, false},
		{"container", WorkloadContainer, false},
		{"STATIC", WorkloadStatic, false},
		{" compose ", WorkloadCompose, false},
		{"kubernetes", "", true},
	}
	for _, tc := range cases {
		gf := &GareFile{Type: tc.typ}
		got, err := gf.ResolveWorkload()
		if (err != nil) != tc.wantErr {
			t.Errorf("ResolveWorkload(%q): err %v, wantErr %v", tc.typ, err, tc.wantErr)
			continue
		}
		if got != tc.want {
			t.Errorf("ResolveWorkload(%q): got %q, want %q", tc.typ, got, tc.want)
		}
	}
}

func TestGareFileResolveComposeFile(t *testing.T) {
	var nilGf *GareFile
	if got := nilGf.ResolveComposeFile(); got != "" {
		t.Errorf("nil ResolveComposeFile: got %q, want empty", got)
	}
	gf := &GareFile{ComposeFile: "  deploy/compose.yml  "}
	if got := gf.ResolveComposeFile(); got != "deploy/compose.yml" {
		t.Errorf("ResolveComposeFile: got %q", got)
	}
}

func TestAppConfigWorkloadPredicates(t *testing.T) {
	var nilCfg *AppConfig
	if nilCfg.IsCompose() || !nilCfg.UsesPodManifest() || nilCfg.UsesAppEnvFile() {
		t.Error("nil config predicates must keep the container defaults")
	}

	cases := []struct {
		appType string
		compose bool
		pod     bool
		appEnv  bool
	}{
		{"", false, true, false},
		{"container", false, true, false},
		{"static", false, false, true},
		{"compose", true, false, true},
		{"COMPOSE", true, false, true},
	}
	for _, tc := range cases {
		assertAppConfigPredicates(t, tc.appType, tc.compose, tc.pod, tc.appEnv)
	}
}

func assertAppConfigPredicates(t *testing.T, appType string, compose, pod, appEnv bool) {
	t.Helper()
	cfg := &AppConfig{AppType: appType}
	if cfg.IsCompose() != compose {
		t.Errorf("IsCompose(%q): got %v, want %v", appType, cfg.IsCompose(), compose)
	}
	if cfg.UsesPodManifest() != pod {
		t.Errorf("UsesPodManifest(%q): got %v, want %v", appType, cfg.UsesPodManifest(), pod)
	}
	if cfg.UsesAppEnvFile() != appEnv {
		t.Errorf("UsesAppEnvFile(%q): got %v, want %v", appType, cfg.UsesAppEnvFile(), appEnv)
	}
}
