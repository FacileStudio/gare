package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FacileStudio/gare/internal/storage"
)

func setupDomainTestApp(t *testing.T, tmpDir, name string, domains []string) *storage.AppConfig {
	t.Helper()
	appDir := filepath.Join(tmpDir, ".local", "share", "gare", "apps", name)
	cfg := &storage.AppConfig{
		Name:    name,
		Port:    8080,
		Domains: domains,
	}
	if err := storage.SaveConfig(appDir, cfg); err != nil {
		t.Fatal(err)
	}
	return cfg
}

func TestDomainAddAndDuplicate(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	setupDomainTestApp(t, tmpDir, "app1", nil)
	setupDomainTestApp(t, tmpDir, "app2", nil)

	cmd := NewDomainCmd()
	cmd.SetArgs([]string{"add", "app1", "app1.example.com"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	appDir := filepath.Join(tmpDir, ".local", "share", "gare", "apps", "app1")
	cfg, err := storage.LoadConfig(appDir)
	if err != nil || len(cfg.Domains) != 1 || cfg.Domains[0] != "app1.example.com" {
		t.Fatalf("expected app1.example.com domain, got: %+v", cfg)
	}

	dupCmd := NewDomainCmd()
	dupCmd.SetArgs([]string{"add", "app1", "app1.example.com"})
	if err := dupCmd.Execute(); err == nil {
		t.Fatal("expected error adding duplicate domain to same app")
	}

	otherCmd := NewDomainCmd()
	otherCmd.SetArgs([]string{"add", "app2", "app1.example.com"})
	if err := otherCmd.Execute(); err == nil {
		t.Fatal("expected error adding domain owned by another app")
	}
}

func TestDomainAddErrors(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	setupDomainTestApp(t, tmpDir, "myapp", nil)

	nonExistCmd := NewDomainCmd()
	nonExistCmd.SetArgs([]string{"add", "ghost", "ghost.example.com"})
	if err := nonExistCmd.Execute(); err == nil {
		t.Fatal("expected error for non-existent app")
	}

	invalidCmd := NewDomainCmd()
	invalidCmd.SetArgs([]string{"add", "myapp", "bad domain with space"})
	if err := invalidCmd.Execute(); err == nil {
		t.Fatal("expected error for invalid domain")
	}
}

func TestDomainRemove(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	setupDomainTestApp(t, tmpDir, "myapp", []string{"a.example.com", "b.example.com"})

	rmCmd := NewDomainCmd()
	rmCmd.SetArgs([]string{"rm", "myapp", "a.example.com"})
	if err := rmCmd.Execute(); err != nil {
		t.Fatalf("unexpected error removing domain: %v", err)
	}

	cfg, _ := storage.LoadConfig(filepath.Join(tmpDir, ".local", "share", "gare", "apps", "myapp"))
	if len(cfg.Domains) != 1 || cfg.Domains[0] != "b.example.com" {
		t.Fatalf("expected b.example.com remaining, got: %+v", cfg.Domains)
	}

	notOwnedCmd := NewDomainCmd()
	notOwnedCmd.SetArgs([]string{"remove", "myapp", "ghost.example.com"})
	if err := notOwnedCmd.Execute(); err == nil {
		t.Fatal("expected error removing unowned domain")
	}
}

func TestDomainListAll(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	setupDomainTestApp(t, tmpDir, "app1", []string{"app1.example.com"})
	setupDomainTestApp(t, tmpDir, "app2", []string{"app2.example.com"})

	var buf bytes.Buffer
	cmd := NewDomainCmd()
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"list", "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var items []domainItem
	if err := json.Unmarshal(buf.Bytes(), &items); err != nil {
		t.Fatalf("invalid json: %v\noutput: %s", err, buf.String())
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 domain items, got: %+v", items)
	}

	buf.Reset()
	cmd.SetArgs([]string{"ls"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "app1.example.com") || !strings.Contains(buf.String(), "app2.example.com") {
		t.Fatalf("expected table to contain domains, got: %s", buf.String())
	}
}

func TestDomainListAppFiltered(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	setupDomainTestApp(t, tmpDir, "app1", []string{"app1.example.com"})

	var buf bytes.Buffer
	cmd := NewDomainCmd()
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"list", "app1", "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var items []domainItem
	if err := json.Unmarshal(buf.Bytes(), &items); err != nil {
		t.Fatalf("invalid json: %v\noutput: %s", err, buf.String())
	}
	if len(items) != 1 || items[0].Hostname != "app1.example.com" || items[0].App != "" {
		t.Fatalf("expected filtered item without App field, got: %+v", items)
	}

	buf.Reset()
	tableCmd := NewDomainCmd()
	tableCmd.SetOut(&buf)
	tableCmd.SetArgs([]string{"list", "app1"})
	if err := tableCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "app1.example.com") {
		t.Fatalf("expected table output to contain domain, got: %s", buf.String())
	}
}

func TestDomainViaAppCmd(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	setupDomainTestApp(t, tmpDir, "testapp", []string{"test.example.com"})

	root := newRootCmd("0.1.0")
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetArgs([]string{"app", "domain", "list", "testapp", "--json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var items []domainItem
	if err := json.Unmarshal(buf.Bytes(), &items); err != nil {
		t.Fatalf("invalid json: %v\noutput: %s", err, buf.String())
	}
	if len(items) != 1 || items[0].Hostname != "test.example.com" {
		t.Fatalf("expected test.example.com, got: %+v", items)
	}
}

func TestSyncAppIngress(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	confDir := filepath.Join(tmpDir, "caddy", "conf.d")
	t.Setenv("GARE_CADDY_CONF_DIR", confDir)
	caddyfilePath := filepath.Join(tmpDir, "caddy", "Caddyfile")
	t.Setenv("GARE_CADDYFILE", caddyfilePath)

	cfg := setupDomainTestApp(t, tmpDir, "capp", []string{"c1.example.com", "c2.example.com"})
	if err := syncAppIngress(cfg); err != nil {
		t.Fatalf("syncAppIngress failed: %v", err)
	}

	snippetPath := filepath.Join(confDir, "capp.caddy")
	data, err := os.ReadFile(snippetPath)
	if err != nil || !strings.Contains(string(data), "c1.example.com, c2.example.com") {
		t.Fatalf("snippet missing expected content: %v, content: %s", err, string(data))
	}

	cfg.Domains = nil
	if err := syncAppIngress(cfg); err != nil {
		t.Fatalf("syncAppIngress remove failed: %v", err)
	}
	if _, err := os.Stat(snippetPath); !os.IsNotExist(err) {
		t.Fatal("expected snippet to be removed when no domains exist")
	}
}
