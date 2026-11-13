package cli

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/FacileStudio/gare/apps/api/internal/deploys"
	"github.com/FacileStudio/gare/apps/api/internal/engine"
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

func TestResolveMaxParallel(t *testing.T) {
	t.Setenv(maxParallelEnv, "")
	if got, err := resolveMaxParallel(); err != nil || got != 2 {
		t.Fatalf("unset: got %d, err %v", got, err)
	}
	t.Setenv(maxParallelEnv, "4")
	if got, err := resolveMaxParallel(); err != nil || got != 4 {
		t.Fatalf("set: got %d, err %v", got, err)
	}
	for _, bad := range []string{"0", "-1", "two"} {
		t.Setenv(maxParallelEnv, bad)
		if _, err := resolveMaxParallel(); err == nil {
			t.Fatalf("expected an error for %q", bad)
		}
	}
}

func TestWebhookDeployCarriesConfigFileSettings(t *testing.T) {
	cfgPath := writeSettingsFile(t, "verbose: true\ncredential_helper: \"!gare-test-helper\"\n")
	root, flags := settingsCommand(t, "--config", cfgPath)
	resolved, err := loadSettings(root, flags)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var received engine.Settings
	var trigger deploys.Trigger
	handler := webhookDeploy(resolved, func(ctx context.Context, app string, tr deploys.Trigger) error {
		received = settingsFrom(ctx)
		trigger = tr
		return nil
	})
	if err := handler(context.Background(), "web"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if received.Auth.CredentialHelper != "!gare-test-helper" || !received.Verbose {
		t.Errorf("a webhook deploy must run with the config file's settings, got %+v", received)
	}
	if trigger != deploys.TriggerWebhook {
		t.Errorf("a webhook deploy must be recorded as one, got trigger %q", trigger)
	}
	if err := handler(context.Background(), "../escape"); err == nil {
		t.Error("expected an invalid app name to be rejected before deploying")
	}
}
