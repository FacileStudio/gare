package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FacileStudio/gare/internal/storage"
)

// rejectingIngress points the Caddy paths at a temporary prefix and puts a caddy on PATH that
// refuses every configuration.
func rejectingIngress(t *testing.T) {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("GARE_CADDY_CONF_DIR", filepath.Join(tmp, "caddy", "conf.d"))
	t.Setenv("GARE_CADDYFILE", filepath.Join(tmp, "caddy", "Caddyfile"))
	rejectingCaddy(t)
}

// rejectingCaddy writes a caddy that exits non-zero and leaves no Caddyfile for the admin API
// fallback to deliver. The reload then fails on a host running a live Caddy exactly as it does on
// one without: the test never hands its temporary Caddyfile to the operator's own ingress.
func rejectingCaddy(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	script := filepath.Join(dir, "caddy")
	body := "#!/bin/sh\nrm -f \"$GARE_CADDYFILE\"\nexit 1\n"
	if err := os.WriteFile(script, []byte(body), 0755); err != nil {
		t.Fatalf("failed to write a rejecting caddy: %v", err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

// TestActivateIngressFailsWhenCaddyRejectsTheConfiguration pins the fatal half of the ingress
// policy: an app meant to be served on its domains cannot be reported active while Caddy refuses
// the configuration, so a refused reload fails the command instead of warning.
func TestActivateIngressFailsWhenCaddyRejectsTheConfiguration(t *testing.T) {
	rejectingIngress(t)
	port := closedPort(t)
	cfg := &storage.AppConfig{
		Name:    "rejected",
		Port:    8000,
		Domains: []string{fmt.Sprintf("rejected.example.com:%d", port)},
	}

	err := activateIngress(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected a rejected configuration to fail the ingress of an app with domains")
	}
	if !strings.Contains(err.Error(), cfg.Name) {
		t.Fatalf("expected the error to name %q, got: %v", cfg.Name, err)
	}
}

// TestActivateIngressWarnsWhenCaddyRejectsADomainlessApp pins the other half: nothing depends on
// ingress for an app with no domains, so the same refusal stays a warning and the command runs on.
func TestActivateIngressWarnsWhenCaddyRejectsADomainlessApp(t *testing.T) {
	rejectingIngress(t)
	cfg := &storage.AppConfig{Name: "plain", Port: 8000}

	if err := activateIngress(context.Background(), cfg); err != nil {
		t.Fatalf("expected a domainless app to warn instead of failing, got: %v", err)
	}
}
