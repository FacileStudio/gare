package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveWebhookSecret(t *testing.T) {
	t.Setenv(webhookSecretEnv, "")
	if _, err := resolveWebhookSecret(""); err == nil {
		t.Fatal("expected an error with no secret anywhere")
	}

	t.Setenv(webhookSecretEnv, " from-env \n")
	if got, err := resolveWebhookSecret(""); err != nil || got != "from-env" {
		t.Fatalf("env secret: got %q, err %v", got, err)
	}

	path := filepath.Join(t.TempDir(), "secret")
	if err := os.WriteFile(path, []byte("from-file\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if got, err := resolveWebhookSecret(path); err != nil || got != "from-file" {
		t.Fatalf("file secret must win over env: got %q, err %v", got, err)
	}

	empty := filepath.Join(t.TempDir(), "empty")
	if err := os.WriteFile(empty, []byte("\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := resolveWebhookSecret(empty); err == nil {
		t.Fatal("expected an error for an empty secret file")
	}
}
