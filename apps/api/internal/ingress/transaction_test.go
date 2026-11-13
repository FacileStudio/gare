package ingress

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/FacileStudio/gare/apps/api/internal/storage"
)

// refusal is the message the fake Caddy refuses a configuration with.
const refusal = "ambiguous site definition: app.example.com"

// fakeReloader counts reloads and refuses the first refused of them, standing in for a Caddy that
// rejects the new snippet and accepts the restored one.
type fakeReloader struct {
	calls   int
	refused int
}

// reload is the Reloader the fake hands to an Ingress.
func (f *fakeReloader) reload(context.Context) error {
	f.calls++
	if f.calls <= f.refused {
		return errors.New(refusal)
	}
	return nil
}

// refusingIngress returns an ingress on a temporary snippet directory whose Caddy refuses the first
// refused reloads and whose probe always answers.
func refusingIngress(t *testing.T, refused int) (*Ingress, *fakeReloader) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("GARE_CADDYFILE", filepath.Join(dir, "Caddyfile"))
	fake := &fakeReloader{refused: refused}
	probe := func(context.Context, Target) error { return nil }
	return &Ingress{ConfDir: dir, Reload: fake.reload, Probe: probe}, fake
}

// assertRolledBack checks the error says the change was rolled back, carries Caddy's message, and
// that Caddy was reloaded a second time with the restored configuration.
func assertRolledBack(t *testing.T, err error, fake *fakeReloader) {
	t.Helper()
	if err == nil || !strings.Contains(err.Error(), "rolled back") || !strings.Contains(err.Error(), refusal) {
		t.Fatalf("expected a rolled-back error carrying caddy's message, got: %v", err)
	}
	if fake.calls != 2 {
		t.Fatalf("expected a reload of the restored configuration after the refusal, got %d reloads", fake.calls)
	}
}

// TestApplyRestoresThePreviousSnippetCaddyRefused pins the rollback: the snippet on disk is the one
// Caddy accepted last, byte for byte, so the next reload by any app does not inherit the refusal.
func TestApplyRestoresThePreviousSnippetCaddyRefused(t *testing.T) {
	ing, fake := refusingIngress(t, 1)
	path := GetSnippetPath(ing.ConfDir, "app")
	previous := []byte("app.example.com {\n\treverse_proxy localhost:8000\n}\n# operator note\n")
	if err := os.WriteFile(path, previous, 0644); err != nil {
		t.Fatal(err)
	}

	app := &storage.AppConfig{Name: "app", Port: 8001, Domains: []string{"app.example.com", "www.example.com"}}
	assertRolledBack(t, ing.Apply(context.Background(), app), fake)
	got, err := os.ReadFile(path)
	if err != nil || string(got) != string(previous) {
		t.Fatalf("expected the previous snippet back byte for byte, got %q (%v)", got, err)
	}
}

// TestApplyRemovesAFirstSnippetCaddyRefused pins the first-time case: there was no snippet, so the
// rollback leaves none.
func TestApplyRemovesAFirstSnippetCaddyRefused(t *testing.T) {
	ing, fake := refusingIngress(t, 1)
	app := &storage.AppConfig{Name: "app", Port: 8001, Domains: []string{"app.example.com"}}

	assertRolledBack(t, ing.Apply(context.Background(), app), fake)
	if _, err := os.Stat(GetSnippetPath(ing.ConfDir, "app")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected the refused first snippet to be removed, got: %v", err)
	}
}

// TestRemovePutsTheSnippetBackWhenCaddyRefuses pins removal as the same transaction: a refused
// configuration without the snippet leaves the snippet in place.
func TestRemovePutsTheSnippetBackWhenCaddyRefuses(t *testing.T) {
	ing, fake := refusingIngress(t, 1)
	path := GetSnippetPath(ing.ConfDir, "app")
	previous := []byte("app.example.com {\n\treverse_proxy localhost:8000\n}\n")
	if err := os.WriteFile(path, previous, 0644); err != nil {
		t.Fatal(err)
	}

	assertRolledBack(t, ing.Remove(context.Background(), "app"), fake)
	got, err := os.ReadFile(path)
	if err != nil || string(got) != string(previous) {
		t.Fatalf("expected the snippet back after a refused removal, got %q (%v)", got, err)
	}
}

// TestApplyReportsARefusedRestore pins the worst case: Caddy refuses the restored configuration
// too, and the error says so instead of claiming a clean rollback.
func TestApplyReportsARefusedRestore(t *testing.T) {
	ing, _ := refusingIngress(t, 2)
	app := &storage.AppConfig{Name: "app", Port: 8001, Domains: []string{"app.example.com"}}

	err := ing.Apply(context.Background(), app)
	if err == nil || !strings.Contains(err.Error(), "refused the restored configuration") {
		t.Fatalf("expected the refused restore to be reported, got: %v", err)
	}
}
