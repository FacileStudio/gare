package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"slices"
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

// openSocket listens on a loopback port and closes every connection it accepts without answering, so
// a caller can tell a bound socket apart from a served hostname.
func openSocket(t *testing.T) (int, net.Listener) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to open a socket: %v", err)
	}
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()
	return listener.Addr().(*net.TCPAddr).Port, listener
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
		t.Fatal("expected activateIngress to fail when nothing answers for the domain port")
	}

	ingress := newTestServer(t, map[string]int{"/": http.StatusOK})
	cfg.Domains = []string{fmt.Sprintf("svc.example.com:%d", ingress.port)}
	if err := activateIngress(ctx, cfg); err != nil {
		t.Fatalf("expected activateIngress to pass with an answering ingress, got: %v", err)
	}

	cfg.Domains = nil
	if err := activateIngress(ctx, cfg); err != nil {
		t.Fatalf("expected a domainless app to skip the ingress probe, got: %v", err)
	}
}

func TestIngressTargets(t *testing.T) {
	defaults := ingressTargets([]string{"app.example.com"})
	want := []ingressTarget{{host: "app.example.com", port: 80}, {host: "app.example.com", port: 443, secure: true}}
	if !slices.Equal(defaults, want) {
		t.Fatalf("expected the standard ingress ports, got: %v", defaults)
	}

	explicit := ingressTargets([]string{"app.example.com:8443", "secure.example.com:443", "plain.example.com:80"})
	want = []ingressTarget{
		{host: "app.example.com", port: 8443},
		{host: "secure.example.com", port: 443, secure: true},
		{host: "plain.example.com", port: 80},
	}
	if !slices.Equal(explicit, want) {
		t.Fatalf("expected every domain to keep its own hostname and port, got: %v", explicit)
	}

	mixed := ingressTargets([]string{"plain.example.com", "custom.example.com:8443"})
	want = []ingressTarget{
		{host: "plain.example.com", port: 80},
		{host: "plain.example.com", port: 443, secure: true},
		{host: "custom.example.com", port: 8443},
	}
	if !slices.Equal(mixed, want) {
		t.Fatalf("expected a hostname on the standard ports beside the explicit one, got: %v", mixed)
	}
}

func TestVerifyIngressSkipsAppsWithoutDomains(t *testing.T) {
	cfg := &storage.AppConfig{Name: "bare", Port: 8000}
	if err := verifyIngress(context.Background(), cfg); err != nil {
		t.Fatalf("expected an app without domains to skip ingress verification, got: %v", err)
	}
}

// TestVerifyIngressRejectsASocketThatDoesNotAnswer pins that a bound port is no longer enough: the
// ingress has to serve the hostname the app is configured for.
func TestVerifyIngressRejectsASocketThatDoesNotAnswer(t *testing.T) {
	port, listener := openSocket(t)
	defer listener.Close()
	cfg := &storage.AppConfig{
		Name:    "stalled",
		Port:    8000,
		Domains: []string{fmt.Sprintf("stalled.example.com:%d", port)},
	}
	if err := verifyIngress(context.Background(), cfg); err == nil {
		t.Fatal("expected a socket that does not answer HTTP to fail ingress verification")
	}
}

// TestVerifyIngressNamesOnlyTheSilentHostname pins a partial failure: an app serving one hostname and
// not the other names the hostname nothing answered for and leaves the answering one out, so the
// report does not send the operator looking at the domain that works.
func TestVerifyIngressNamesOnlyTheSilentHostname(t *testing.T) {
	served := newTestServer(t, map[string]int{"/": http.StatusOK})
	cfg := &storage.AppConfig{
		Name: "partial",
		Port: 8000,
		Domains: []string{
			fmt.Sprintf("served.example.com:%d", served.port),
			fmt.Sprintf("silent.example.com:%d", closedPort(t)),
		},
	}

	err := verifyIngress(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected the silent hostname to fail ingress verification")
	}
	if !strings.Contains(err.Error(), "silent.example.com") {
		t.Fatalf("expected the error to name the silent hostname, got: %v", err)
	}
	if strings.Contains(err.Error(), "served.example.com") {
		t.Fatalf("expected the error to leave the answering hostname out, got: %v", err)
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
