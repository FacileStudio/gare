package compose

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestDetectFile(t *testing.T) {
	repoDir := t.TempDir()
	if got := DetectFile(repoDir); got != "" {
		t.Errorf("DetectFile on empty repo: got %q, want empty", got)
	}
	writeComposeFile(t, repoDir, "docker-compose.yml")
	if got := DetectFile(repoDir); got != "docker-compose.yml" {
		t.Errorf("DetectFile: got %q, want docker-compose.yml", got)
	}
	writeComposeFile(t, repoDir, "compose.yaml")
	if got := DetectFile(repoDir); got != "compose.yaml" {
		t.Errorf("DetectFile precedence: got %q, want compose.yaml", got)
	}
}

func TestLocateFile(t *testing.T) {
	repoDir := t.TempDir()
	if _, err := LocateFile(repoDir, ""); err == nil {
		t.Error("expected error when no compose file exists")
	}
	writeComposeFile(t, repoDir, "docker-compose.yml")
	got, err := LocateFile(repoDir, "")
	if err != nil || got != "docker-compose.yml" {
		t.Errorf("LocateFile: got %q, err %v", got, err)
	}
	got, err = LocateFile(repoDir, "docker-compose.yml")
	if err != nil || got != "docker-compose.yml" {
		t.Errorf("LocateFile explicit: got %q, err %v", got, err)
	}
	testLocateFileRejects(t, repoDir)
}

func testLocateFileRejects(t *testing.T, repoDir string) {
	invalid := []string{"missing.yml", "with space.yml", "../escape.yml", "/etc/compose.yml"}
	for _, name := range invalid {
		if _, err := LocateFile(repoDir, name); err == nil {
			t.Errorf("expected error for compose file %q", name)
		}
	}
}

func TestPublishedPorts(t *testing.T) {
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

	ports, err := PublishedPorts(filepath.Join(repoDir, "compose.yml"))
	if err != nil {
		t.Fatalf("PublishedPorts failed: %v", err)
	}
	want := []int{1000, 5432, 8000, 9000, 9100}
	slices.Sort(ports)
	if !slices.Equal(ports, want) {
		t.Errorf("PublishedPorts: got %v, want %v", ports, want)
	}
}

func TestPublishedPortsInvalid(t *testing.T) {
	repoDir := t.TempDir()
	if _, err := PublishedPorts(filepath.Join(repoDir, "missing.yml")); err == nil {
		t.Error("expected error for missing compose file")
	}
	writeComposeFile(t, repoDir, "broken.yml", "services:\n  web:\n    ports: [")
	if _, err := PublishedPorts(filepath.Join(repoDir, "broken.yml")); err == nil {
		t.Error("expected error for invalid compose yaml")
	}
}

func TestValidatePort(t *testing.T) {
	repoDir := t.TempDir()
	writeComposeFile(t, repoDir, "compose.yml", "services:\n  web:\n    ports:\n      - \"8200:80\"\n")
	composePath := filepath.Join(repoDir, "compose.yml")

	if err := ValidatePort(composePath, 8200); err != nil {
		t.Errorf("ValidatePort on published port: unexpected error %v", err)
	}
	if err := ValidatePort(composePath, 8300); err == nil {
		t.Error("expected error for unpublished port")
	}
	if err := ValidatePort(composePath, 0); err == nil {
		t.Error("expected error for missing port")
	}
}

func TestValidatePortWithoutMappings(t *testing.T) {
	repoDir := t.TempDir()
	writeComposeFile(t, repoDir, "compose.yml", "services:\n  worker:\n    image: busybox\n")
	if err := ValidatePort(filepath.Join(repoDir, "compose.yml"), 8200); err == nil {
		t.Error("expected error when the compose file publishes no host ports")
	}
}

func TestPortPair(t *testing.T) {
	cases := []struct {
		entry     any
		published int
		target    int
	}{
		{"8000:80", 8000, 80},
		{"8000", 8000, 8000},
		{"127.0.0.1:9100:91", 9100, 91},
		{"1000-1010:1000", 1000, 1000},
		{"8080:80/udp", 8080, 80},
		{9000, 9000, 9000},
		{map[string]any{"published": 18420, "target": 80}, 18420, 80},
	}
	for _, tc := range cases {
		published, target := portPair(tc.entry)
		if published != tc.published || target != tc.target {
			t.Errorf("portPair(%v): got %d:%d, want %d:%d", tc.entry, published, target, tc.published, tc.target)
		}
	}
}

func TestLoopbackOnly(t *testing.T) {
	cases := []struct {
		name    string
		content string
		want    bool
	}{
		{"loopback", "services:\n  api:\n    ports:\n      - \"127.0.0.1:4000:4000\"\n", true},
		{"localhost", "services:\n  api:\n    ports:\n      - \"localhost:4000:4000\"\n", true},
		{"ipv6-loopback", "services:\n  api:\n    ports:\n      - \"[::1]:4000:4000\"\n", true},
		{"all-interfaces", "services:\n  api:\n    ports:\n      - \"4000:4000\"\n", false},
		{"mixed", "services:\n  api:\n    ports:\n      - \"127.0.0.1:4000:4000\"\n      - \"8000:80\"\n", false},
		{"long-syntax-loopback", "services:\n  api:\n    ports:\n" +
			"      - target: 4000\n        published: 4000\n        host_ip: \"127.0.0.1\"\n", true},
		{"unpublished", "services:\n  worker:\n    image: busybox\n", false},
	}
	for _, tc := range cases {
		repoDir := t.TempDir()
		writeComposeFile(t, repoDir, "compose.yml", tc.content)
		got, err := LoopbackOnly(filepath.Join(repoDir, "compose.yml"))
		if err != nil {
			t.Fatalf("%s: LoopbackOnly failed: %v", tc.name, err)
		}
		if got != tc.want {
			t.Errorf("%s: got %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestLoopbackOnlyInvalidFile(t *testing.T) {
	repoDir := t.TempDir()
	if _, err := LoopbackOnly(filepath.Join(repoDir, "missing.yml")); err == nil {
		t.Error("expected error for a missing compose file")
	}
	writeComposeFile(t, repoDir, "broken.yml", "services:\n  web:\n    ports: [")
	if _, err := LoopbackOnly(filepath.Join(repoDir, "broken.yml")); err == nil {
		t.Error("expected error for invalid compose yaml")
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
