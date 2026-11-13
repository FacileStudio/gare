package cli

import (
	"bytes"
	"testing"
)

func TestAppCreateCmdFlagParsingNoPanic(t *testing.T) {
	root := newRootCmd("0.1.0")
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"app", "create", "--help"})
	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error running help: %v", err)
	}
}

func TestAppCreateCmdPortFlagShorthand(t *testing.T) {
	root := newRootCmd("0.1.0")
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"app", "create", "testapp", "--repo", "https://github.com/test/repo", "-p", "8080", "--help"})
	if err := root.Execute(); err != nil {
		t.Fatalf("unexpected error executing create with -p shorthand: %v", err)
	}
}
