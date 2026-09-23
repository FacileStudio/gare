package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/FacileStudio/gare/internal/storage"
)

func fakeCaddy(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	script := filepath.Join(dir, "caddy")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
		t.Fatalf("failed to write a fake caddy: %v", err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func TestActivateIngress(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("GARE_CADDY_CONF_DIR", filepath.Join(tmp, "caddy", "conf.d"))
	t.Setenv("GARE_CADDYFILE", filepath.Join(tmp, "caddy", "Caddyfile"))
	fakeCaddy(t)
	ctx := context.Background()

	cfg := &storage.AppConfig{Name: "svc", Port: 8000, Domains: []string{fmt.Sprintf("svc.example.com:%d", closedPort(t))}}
	if err := activateIngress(ctx, cfg); err == nil {
		t.Fatal("expected activateIngress to fail when nothing listens on the domain port")
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to open a listener: %v", err)
	}
	defer listener.Close()
	cfg.Domains = []string{fmt.Sprintf("svc.example.com:%d", listener.Addr().(*net.TCPAddr).Port)}
	if err := activateIngress(ctx, cfg); err != nil {
		t.Fatalf("expected activateIngress to pass with a listening ingress, got: %v", err)
	}

	cfg.Domains = nil
	if err := activateIngress(ctx, cfg); err != nil {
		t.Fatalf("expected a domainless app to skip the ingress probe, got: %v", err)
	}
}

func TestIngressPorts(t *testing.T) {
	defaults := ingressPorts([]string{"app.example.com"})
	if len(defaults) != 2 || defaults[0] != 80 || defaults[1] != 443 {
		t.Fatalf("expected the standard ingress ports, got: %v", defaults)
	}

	explicit := ingressPorts([]string{"app.example.com:8443", "other.example.com:8443", "third.example.com:8080"})
	if len(explicit) != 2 || explicit[0] != 8443 || explicit[1] != 8080 {
		t.Fatalf("expected deduplicated explicit ports in order, got: %v", explicit)
	}
}

func TestVerifyIngressSkipsAppsWithoutDomains(t *testing.T) {
	cfg := &storage.AppConfig{Name: "bare", Port: 8000}
	if err := verifyIngress(context.Background(), cfg); err != nil {
		t.Fatalf("expected an app without domains to skip ingress verification, got: %v", err)
	}
}

func TestVerifyIngressReportsMissingListener(t *testing.T) {
	port := closedPort(t)
	cfg := &storage.AppConfig{
		Name:    "broken",
		Port:    8000,
		Domains: []string{fmt.Sprintf("broken.example.com:%d", port)},
	}
	err := verifyIngress(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected ingress verification to fail with no listener")
	}
	for _, needle := range []string{"broken", "broken.example.com", strconv.Itoa(port)} {
		if !strings.Contains(err.Error(), needle) {
			t.Fatalf("expected the error to mention %q, got: %v", needle, err)
		}
	}
}
