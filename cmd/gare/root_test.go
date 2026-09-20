package main

import (
	"os"
	"testing"
)

func TestRootRegistersNoColorFlag(t *testing.T) {
	root := newRootCmd("test")
	if root.PersistentFlags().Lookup("no-color") == nil {
		t.Fatal("root must register --no-color as a persistent flag")
	}
	if root.PersistentPreRunE == nil {
		t.Fatal("root must use PersistentPreRunE so subcommands apply the flags")
	}
	if root.PreRunE != nil {
		t.Error("root must not rely on PreRunE, which never runs for subcommands")
	}
}

func TestApplyColorPreference(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	if err := applyColorPreference(false); err != nil {
		t.Fatalf("plain run must not fail: %v", err)
	}
	if os.Getenv("NO_COLOR") != "" {
		t.Errorf("colors must stay enabled without --no-color")
	}

	if err := applyColorPreference(true); err != nil {
		t.Fatalf("applying --no-color failed: %v", err)
	}
	if os.Getenv("NO_COLOR") == "" {
		t.Errorf("--no-color must export NO_COLOR")
	}
}
