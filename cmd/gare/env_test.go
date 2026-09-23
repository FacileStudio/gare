package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/FacileStudio/gare/internal/manifest"
	"github.com/FacileStudio/gare/internal/storage"
)

func setupTestApp(t *testing.T, name string, isStatic bool) (string, string) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("XDG_DATA_HOME", "")
	appDir := filepath.Join(tmpDir, ".local", "share", "gare", "apps", name)
	appType := "container"
	if isStatic {
		appType = "static"
	}
	cfg := &storage.AppConfig{
		Name:    name,
		Port:    8080,
		AppType: appType,
	}
	if err := storage.SaveConfig(appDir, cfg); err != nil {
		t.Fatal(err)
	}
	manifestPath := storage.GetManifestPath(appDir)
	if !isStatic {
		if err := manifest.Generate(name, 8080, 8080, manifestPath); err != nil {
			t.Fatal(err)
		}
	}
	return tmpDir, appDir
}

func TestEnvSetAndList(t *testing.T) {
	_, appDir := setupTestApp(t, "testenv", false)
	cmdSet := NewEnvCmd()
	cmdSet.SetArgs([]string{"set", "testenv", "KEY1=VAL1", "KEY2=VAL2"})
	if err := cmdSet.Execute(); err != nil {
		t.Fatalf("env set failed: %v", err)
	}

	manifestPath := storage.GetManifestPath(appDir)
	envs, err := manifest.GetEnv(manifestPath)
	if err != nil || len(envs) != 2 || envs["KEY1"] != "VAL1" {
		t.Fatalf("unexpected manifest envs: %+v, err: %v", envs, err)
	}

	cmdList := NewEnvCmd()
	var buf bytes.Buffer
	cmdList.SetOut(&buf)
	cmdList.SetArgs([]string{"list", "testenv", "--json"})
	if err := cmdList.Execute(); err != nil {
		t.Fatalf("env list failed: %v", err)
	}

	var jsonMap map[string]string
	if err := json.Unmarshal(buf.Bytes(), &jsonMap); err != nil {
		t.Fatalf("invalid json output: %v", err)
	}
	if jsonMap["KEY2"] != "VAL2" {
		t.Errorf("expected KEY2=VAL2, got %s", jsonMap["KEY2"])
	}
}

func TestEnvUnset(t *testing.T) {
	_, appDir := setupTestApp(t, "testenv2", false)
	manifestPath := storage.GetManifestPath(appDir)
	if err := manifest.SetEnv(manifestPath, map[string]string{"K1": "V1", "K2": "V2"}); err != nil {
		t.Fatal(err)
	}

	cmdUnset := NewEnvCmd()
	cmdUnset.SetArgs([]string{"unset", "testenv2", "K1"})
	if err := cmdUnset.Execute(); err != nil {
		t.Fatalf("env unset failed: %v", err)
	}

	envs, _ := manifest.GetEnv(manifestPath)
	if _, ok := envs["K1"]; ok {
		t.Fatalf("expected K1 to be unset, got %+v", envs)
	}
}

func TestEnvLoad(t *testing.T) {
	tmpDir, appDir := setupTestApp(t, "testenv3", false)
	envFile := filepath.Join(tmpDir, ".env")
	if err := os.WriteFile(envFile, []byte("FROM_FILE=loaded\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cmdLoad := NewEnvCmd()
	cmdLoad.SetArgs([]string{"load", "testenv3", "-f", envFile})
	if err := cmdLoad.Execute(); err != nil {
		t.Fatalf("env load failed: %v", err)
	}

	manifestPath := storage.GetManifestPath(appDir)
	envs, _ := manifest.GetEnv(manifestPath)
	if envs["FROM_FILE"] != "loaded" {
		t.Errorf("expected FROM_FILE=loaded, got %s", envs["FROM_FILE"])
	}
}

func TestEnvStaticApp(t *testing.T) {
	_, appDir := setupTestApp(t, "staticapp", true)
	cmdSet := NewEnvCmd()
	cmdSet.SetArgs([]string{"set", "staticapp", "FOO=BAR"})
	if err := cmdSet.Execute(); err != nil {
		t.Fatalf("expected static app env set to succeed, got: %v", err)
	}
	envs, err := storage.GetAppEnv(appDir)
	if err != nil || envs["FOO"] != "BAR" {
		t.Fatalf("expected FOO=BAR in static env, got: %+v, err: %v", envs, err)
	}

	cmdUnset := NewEnvCmd()
	cmdUnset.SetArgs([]string{"unset", "staticapp", "FOO"})
	if err := cmdUnset.Execute(); err != nil {
		t.Fatalf("expected static app env unset to succeed, got: %v", err)
	}
	envs, _ = storage.GetAppEnv(appDir)
	if _, ok := envs["FOO"]; ok {
		t.Fatalf("expected FOO to be unset, got: %+v", envs)
	}
}
