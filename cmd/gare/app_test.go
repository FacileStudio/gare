package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FacileStudio/gare/internal/storage"
)

func TestValidateCreateInputs(t *testing.T) {
	testValidInputs(t)
	testInvalidNames(t)
	testInvalidDomains(t)
	testMissingRequired(t)
}

func testValidInputs(t *testing.T) {
	valid := []struct {
		name   string
		domain string
	}{
		{"myapp", "example.com"},
		{"my-app-1", "sub.example.com"},
		{"app_test", "localhost"},
		{"a", "localhost:8080"},
		{"web-service", "*.example.com"},
	}
	for _, tc := range valid {
		opts := appCreateOptions{repo: "https://git.example.com/repo", domain: tc.domain}
		if err := validateCreateInputs(tc.name, opts); err != nil {
			t.Errorf("expected %s / %s to be valid, got: %v", tc.name, tc.domain, err)
		}
	}
}

func testInvalidNames(t *testing.T) {
	invalidNames := []string{
		"../traversal",
		"bad/slash",
		"has space",
		"-startshyphen",
		"_startsunderscore",
		"",
		strings.Repeat("a", 64),
		"bad$char",
	}
	opts := appCreateOptions{repo: "https://git.example.com/repo", domain: "example.com"}
	for _, name := range invalidNames {
		if err := validateCreateInputs(name, opts); err == nil {
			t.Errorf("expected invalid name %q to fail validation", name)
		}
	}
}

func testInvalidDomains(t *testing.T) {
	invalidDomains := []string{
		"example.com { reverse_proxy }",
		"example.com\nnewline",
		"bad;injection",
		"domain/path",
		"has space.com",
		"{injection}",
		"bad#comment",
	}
	for _, domain := range invalidDomains {
		opts := appCreateOptions{repo: "https://git.example.com/repo", domain: domain}
		if err := validateCreateInputs("myapp", opts); err == nil {
			t.Errorf("expected invalid domain %q to fail validation", domain)
		}
	}
}

func testMissingRequired(t *testing.T) {
	if err := validateCreateInputs("myapp", appCreateOptions{domain: "example.com"}); err == nil {
		t.Error("expected error when --repo is empty")
	}
}

func TestRootCmdVersion(t *testing.T) {
	cmd := NewRootCmd("0.1.0")
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"--version"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "gare 0.1.0\n"
	if buf.String() != expected {
		t.Errorf("expected %q, got %q", expected, buf.String())
	}
	if !cmd.SilenceErrors {
		t.Error("expected SilenceErrors to be true")
	}
}

func TestListCmdJSON(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	appDir := filepath.Join(tmpDir, ".local", "share", "gare", "apps", "testapp")
	cfg := &storage.AppConfig{Name: "testapp", Port: 8000, Domain: "test.local"}
	if err := storage.SaveConfig(appDir, cfg); err != nil {
		t.Fatal(err)
	}
	cmd := NewListCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var items []appListItem
	if err := json.Unmarshal(buf.Bytes(), &items); err != nil {
		t.Fatalf("invalid json: %v\noutput: %s", err, buf.String())
	}
	if len(items) != 1 || items[0].Name != "testapp" {
		t.Errorf("expected 1 item with name testapp, got: %+v", items)
	}
}

func TestListCmdQuiet(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	appDir := filepath.Join(tmpDir, ".local", "share", "gare", "apps", "quietapp")
	cfg := &storage.AppConfig{Name: "quietapp", Port: 8001, Domain: "quiet.local"}
	if err := storage.SaveConfig(appDir, cfg); err != nil {
		t.Fatal(err)
	}
	cmd := NewListCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"-q"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "quietapp") {
		t.Errorf("expected output to contain quietapp, got %q", buf.String())
	}
}
