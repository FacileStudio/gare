package main

import (
	"os"
	"testing"

	"github.com/FacileStudio/gare/internal/storage"
)

func TestResolveDestroyWorkloadReadsTheConfiguredType(t *testing.T) {
	cases := []struct {
		label   string
		config  string
		want    storage.WorkloadType
		wantErr bool
	}{
		{"compose", `{"name":"myapp","app_type":"compose"}`, storage.WorkloadCompose, false},
		{"static", `{"name":"myapp","app_type":"static"}`, storage.WorkloadStatic, false},
		{"container", `{"name":"myapp","app_type":"container"}`, storage.WorkloadContainer, false},
		{"unset defaults to container", `{"name":"myapp"}`, storage.WorkloadContainer, false},
		{"unknown type", `{"name":"myapp","app_type":"kubernetes"}`, storage.WorkloadUnknown, true},
	}
	for _, tc := range cases {
		t.Run(tc.label, func(t *testing.T) {
			appDir := t.TempDir()
			if err := os.WriteFile(storage.GetConfigPath(appDir), []byte(tc.config), 0644); err != nil {
				t.Fatal(err)
			}

			cfg, workload, err := resolveDestroyWorkload(appDir)
			if (err != nil) != tc.wantErr {
				t.Fatalf("resolveDestroyWorkload error = %v, wantErr %v", err, tc.wantErr)
			}
			if workload != tc.want {
				t.Errorf("workload = %q, want %q", workload, tc.want)
			}
			if tc.wantErr && cfg != nil {
				t.Errorf("config = %+v, want nil alongside an unresolved workload", cfg)
			}
		})
	}
}

func TestResolveDestroyWorkloadRefusesToGuessAnUnreadableConfig(t *testing.T) {
	cfg, workload, err := resolveDestroyWorkload(t.TempDir())
	if err == nil {
		t.Fatal("expected an unreadable config to be reported")
	}
	if workload != storage.WorkloadUnknown {
		t.Errorf("workload = %q, want unknown rather than a container default", workload)
	}
	if cfg != nil {
		t.Errorf("config = %+v, want nil", cfg)
	}
}

func TestResolveDestroyWorkloadRejectsAMalformedConfig(t *testing.T) {
	appDir := t.TempDir()
	if err := os.WriteFile(storage.GetConfigPath(appDir), []byte("{not json"), 0644); err != nil {
		t.Fatal(err)
	}

	_, workload, err := resolveDestroyWorkload(appDir)
	if err == nil {
		t.Fatal("expected a malformed config to be reported")
	}
	if workload != storage.WorkloadUnknown {
		t.Errorf("workload = %q, want unknown rather than a container default", workload)
	}
}
