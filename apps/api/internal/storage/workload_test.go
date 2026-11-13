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

func TestAppConfigResolveWorkloadType(t *testing.T) {
	var nilCfg *AppConfig
	if got, err := nilCfg.ResolveWorkloadType(); err == nil || got != WorkloadUnknown {
		t.Errorf("nil ResolveWorkloadType: got %q, err %v, want unknown and an error", got, err)
	}

	cases := []struct {
		appType string
		want    WorkloadType
		wantErr bool
	}{
		{"", WorkloadContainer, false},
		{"container", WorkloadContainer, false},
		{"STATIC", WorkloadStatic, false},
		{" compose ", WorkloadCompose, false},
		{"kubernetes", WorkloadUnknown, true},
	}
	for _, tc := range cases {
		cfg := &AppConfig{AppType: tc.appType}
		got, err := cfg.ResolveWorkloadType()
		if (err != nil) != tc.wantErr {
			t.Errorf("ResolveWorkloadType(%q): err %v, wantErr %v", tc.appType, err, tc.wantErr)
			continue
		}
		if got != tc.want {
			t.Errorf("ResolveWorkloadType(%q): got %q, want %q", tc.appType, got, tc.want)
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
