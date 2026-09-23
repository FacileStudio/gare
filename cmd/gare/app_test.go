package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/FacileStudio/gare/internal/storage"
	"github.com/spf13/cobra"
)

func TestValidateCreateInputs(t *testing.T) {
	testValidInputs(t)
	testInvalidNames(t)
	testMissingRequired(t)
}

func testValidInputs(t *testing.T) {
	valid := []string{
		"myapp",
		"my-app-1",
		"app_test",
		"a",
		"web-service",
	}
	for _, name := range valid {
		opts := appCreateOptions{repo: "https://git.example.com/repo"}
		if err := validateCreateInputs(name, opts); err != nil {
			t.Errorf("expected %s to be valid, got: %v", name, err)
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
	opts := appCreateOptions{repo: "https://git.example.com/repo"}
	for _, name := range invalidNames {
		if err := validateCreateInputs(name, opts); err == nil {
			t.Errorf("expected invalid name %q to fail validation", name)
		}
	}
}

func testMissingRequired(t *testing.T) {
	if err := validateCreateInputs("myapp", appCreateOptions{}); err == nil {
		t.Error("expected error when --repo is empty")
	}
}

func TestRootCmdVersion(t *testing.T) {
	cmd := newRootCmd("0.1.0")
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
	cfg := &storage.AppConfig{Name: "testapp", Port: 8000, Domains: []string{"test.local"}}
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
	cfg := &storage.AppConfig{Name: "quietapp", Port: 8001, Domains: []string{"quiet.local"}}
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

func TestAppSubcommands(t *testing.T) {
	appCmd := NewAppCmd()
	expected := []string{
		"create", "deploy", "domain", "list", "status", "start",
		"stop", "restart", "logs", "destroy", "env",
	}
	for _, name := range expected {
		if findSubcommand(appCmd, name) == nil {
			t.Errorf("expected app subcommand %q not found", name)
		}
	}
}

func TestCommandAliases(t *testing.T) {
	if !slices.Contains(NewAppCmd().Aliases, "apps") {
		t.Error("expected app command to have 'apps' alias")
	}
	for _, alias := range []string{"delete", "rm"} {
		if !slices.Contains(NewDestroyCmd().Aliases, alias) {
			t.Errorf("expected destroy command to have alias %q", alias)
		}
	}
	for _, alias := range []string{"ls", "ps"} {
		if !slices.Contains(NewListCmd().Aliases, alias) {
			t.Errorf("expected list command to have alias %q", alias)
		}
	}
}

func findSubcommand(parent *cobra.Command, name string) *cobra.Command {
	for _, sub := range parent.Commands() {
		if sub.Name() == name {
			return sub
		}
	}
	return nil
}

func TestAppListExecutionViaAppCmd(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	appDir := filepath.Join(tmpDir, ".local", "share", "gare", "apps", "subapp")
	cfg := &storage.AppConfig{Name: "subapp", Port: 8002, Domains: []string{"sub.local"}}
	if err := storage.SaveConfig(appDir, cfg); err != nil {
		t.Fatal(err)
	}

	root := newRootCmd("0.1.0")
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetArgs([]string{"app", "list", "--json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var items []appListItem
	if err := json.Unmarshal(buf.Bytes(), &items); err != nil {
		t.Fatalf("invalid json: %v\noutput: %s", err, buf.String())
	}
	if len(items) != 1 || items[0].Name != "subapp" {
		t.Errorf("expected 1 item with name subapp, got: %+v", items)
	}
}
