package podman

import (
	"os"
	"path/filepath"
	"testing"
)

func runExposeCase(t *testing.T, content string, expected int) {
	t.Helper()
	tmpDir := t.TempDir()
	cfPath := filepath.Join(tmpDir, "Containerfile")
	if err := os.WriteFile(cfPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}
	if got := ParseExposedPort(cfPath); got != expected {
		t.Errorf("ParseExposedPort() = %d, want %d", got, expected)
	}
}

func TestParseExposedPortValid(t *testing.T) {
	cases := []struct {
		content  string
		expected int
	}{
		{"FROM alpine\nEXPOSE 8080\nCMD [\"sleep\", \"10\"]\n", 8080},
		{"FROM golang\nEXPOSE 3000/tcp\n", 3000},
		{"FROM node\nexpose 5000\n", 5000},
		{"FROM nginx\nEXPOSE 80 443\n", 80},
		{"FROM alpine\nEXPOSE\t8082\n", 8082},
	}
	for _, tc := range cases {
		runExposeCase(t, tc.content, tc.expected)
	}
}

func TestParseExposedPortInvalid(t *testing.T) {
	cases := []struct {
		content  string
		expected int
	}{
		{"# EXPOSE 9999\nEXPOSE 8000\n", 8000},
		{"FROM scratch\nCOPY app /\n", 0},
		{"FROM alpine\nEXPOSE notanumber\n", 0},
		{"FROM alpine\nEXPOSE 70000\n", 0},
	}
	for _, tc := range cases {
		runExposeCase(t, tc.content, tc.expected)
	}
}

func TestDetectExposedPort(t *testing.T) {
	tmpDir := t.TempDir()
	cfPath := filepath.Join(tmpDir, "Dockerfile")
	if err := os.WriteFile(cfPath, []byte("FROM golang:alpine\nEXPOSE 8081\n"), 0644); err != nil {
		t.Fatalf("failed to write Dockerfile: %v", err)
	}
	if got := DetectExposedPort(tmpDir, ""); got != 8081 {
		t.Errorf("DetectExposedPort auto = %d, want 8081", got)
	}

	customCf := filepath.Join(tmpDir, "custom.Containerfile")
	if err := os.WriteFile(customCf, []byte("FROM alpine\nEXPOSE 9090\n"), 0644); err != nil {
		t.Fatalf("failed to write custom Containerfile: %v", err)
	}
	if got := DetectExposedPort(tmpDir, "custom.Containerfile"); got != 9090 {
		t.Errorf("DetectExposedPort custom = %d, want 9090", got)
	}
}
