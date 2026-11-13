package cli

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/FacileStudio/gare/apps/api/internal/storage"
)

func TestRenderCreatedAtUsesTheLocalTimezone(t *testing.T) {
	if _, offset := time.Now().Zone(); offset == 0 {
		t.Skip("host runs in UTC, where the stored timestamp and its local rendering coincide")
	}
	if got := renderCreatedAt("2026-09-23T02:44:52Z"); got == "2026-09-23 02:44" {
		t.Errorf("expected the stored UTC timestamp in local time, got %s", got)
	}
}

func TestRenderCreatedAtFallsBackToTheStoredValue(t *testing.T) {
	if got := renderCreatedAt("not-a-timestamp"); got != "not-a-timestamp" {
		t.Errorf("expected an unparseable timestamp to be shown as stored, got %q", got)
	}
}

func TestListCmdStaticTable(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	appDir := filepath.Join(tmpDir, ".local", "share", "gare", "apps", "staticapp")
	cfg := &storage.AppConfig{
		Name:    "staticapp",
		Port:    0,
		Domains: []string{"static.local"},
		AppType: "static",
	}
	if err := storage.Open(filepath.Dir(appDir)).Save(cfg); err != nil {
		t.Fatal(err)
	}
	cmd := NewListCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "staticapp") {
		t.Errorf("expected output to contain staticapp, got %q", output)
	}
	if !strings.Contains(output, "static") {
		t.Errorf("expected output to display static status, got %q", output)
	}
}
