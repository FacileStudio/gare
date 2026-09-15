package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestLogsCmdFlags(t *testing.T) {
	cmd := NewLogsCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "-n, --lines") {
		t.Errorf("expected logs help to include -n / --lines flag, got %q", output)
	}
	if !strings.Contains(output, "-f, --follow") {
		t.Errorf("expected logs help to include -f / --follow flag, got %q", output)
	}
}

func TestListCmdEmptyTable(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	cmd := NewListCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := strings.TrimSpace(buf.String())
	if output != "No applications configured" {
		t.Errorf("got %q, want %q", output, "No applications configured")
	}
}
