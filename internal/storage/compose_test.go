package storage

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestDetectComposeFile(t *testing.T) {
	repoDir := t.TempDir()
	if got := DetectComposeFile(repoDir); got != "" {
		t.Errorf("DetectComposeFile on empty repo: got %q, want empty", got)
	}
	writeComposeFile(t, repoDir, "docker-compose.yml")
	if got := DetectComposeFile(repoDir); got != "docker-compose.yml" {
		t.Errorf("DetectComposeFile: got %q, want docker-compose.yml", got)
	}
	writeComposeFile(t, repoDir, "compose.yaml")
	if got := DetectComposeFile(repoDir); got != "compose.yaml" {
		t.Errorf("DetectComposeFile precedence: got %q, want compose.yaml", got)
	}
}

func TestLocateComposeFile(t *testing.T) {
	repoDir := t.TempDir()
	if _, err := LocateComposeFile(repoDir, ""); err == nil {
		t.Error("expected error when no compose file exists")
	}
	writeComposeFile(t, repoDir, "docker-compose.yml")
	got, err := LocateComposeFile(repoDir, "")
	if err != nil || got != "docker-compose.yml" {
		t.Errorf("LocateComposeFile: got %q, err %v", got, err)
	}
	got, err = LocateComposeFile(repoDir, "docker-compose.yml")
	if err != nil || got != "docker-compose.yml" {
		t.Errorf("LocateComposeFile explicit: got %q, err %v", got, err)
	}
	testLocateComposeFileRejects(t, repoDir)
}

func testLocateComposeFileRejects(t *testing.T, repoDir string) {
	invalid := []string{"missing.yml", "with space.yml", "../escape.yml", "/etc/compose.yml"}
	for _, name := range invalid {
		if _, err := LocateComposeFile(repoDir, name); err == nil {
			t.Errorf("expected error for compose file %q", name)
		}
	}
}

func TestComposePublishedPorts(t *testing.T) {
	repoDir := t.TempDir()
	content := "services:\n" +
		"  web:\n" +
		"    image: nginx\n" +
		"    ports:\n" +
		"      - \"8000:80\"\n" +
		"      - 9000\n" +
		"      - \"127.0.0.1:9100:91\"\n" +
		"      - \"1000-1010:1000\"\n" +
		"  db:\n" +
		"    ports:\n" +
		"      - target: 5432\n" +
		"        published: 5432\n"
	writeComposeFile(t, repoDir, "compose.yml", content)

	ports, err := ComposePublishedPorts(filepath.Join(repoDir, "compose.yml"))
	if err != nil {
		t.Fatalf("ComposePublishedPorts failed: %v", err)
	}
	want := []int{1000, 5432, 8000, 9000, 9100}
	slices.Sort(ports)
	if !slices.Equal(ports, want) {
		t.Errorf("ComposePublishedPorts: got %v, want %v", ports, want)
	}
}

func TestComposePublishedPortsInvalid(t *testing.T) {
	repoDir := t.TempDir()
	if _, err := ComposePublishedPorts(filepath.Join(repoDir, "missing.yml")); err == nil {
		t.Error("expected error for missing compose file")
	}
	writeComposeFile(t, repoDir, "broken.yml", "services:\n  web:\n    ports: [")
	if _, err := ComposePublishedPorts(filepath.Join(repoDir, "broken.yml")); err == nil {
		t.Error("expected error for invalid compose yaml")
	}
}

func TestValidateComposePort(t *testing.T) {
	repoDir := t.TempDir()
	writeComposeFile(t, repoDir, "compose.yml", "services:\n  web:\n    ports:\n      - \"8200:80\"\n")
	composePath := filepath.Join(repoDir, "compose.yml")

	if err := ValidateComposePort(composePath, 8200); err != nil {
		t.Errorf("ValidateComposePort on published port: unexpected error %v", err)
	}
	if err := ValidateComposePort(composePath, 8300); err == nil {
		t.Error("expected error for unpublished port")
	}
	if err := ValidateComposePort(composePath, 0); err == nil {
		t.Error("expected error for missing port")
	}
}

func TestValidateComposePortWithoutMappings(t *testing.T) {
	repoDir := t.TempDir()
	writeComposeFile(t, repoDir, "compose.yml", "services:\n  worker:\n    image: busybox\n")
	err := ValidateComposePort(filepath.Join(repoDir, "compose.yml"), 8200)
	if err == nil {
		t.Error("expected error when the compose file publishes no host ports")
	}
}

func writeComposeFile(t *testing.T, repoDir, name string, content ...string) {
	t.Helper()
	body := "services:\n  web:\n    image: nginx\n"
	if len(content) > 0 {
		body = content[0]
	}
	if err := os.WriteFile(filepath.Join(repoDir, name), []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
}
