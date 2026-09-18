package storage_test

import (
	"testing"

	"github.com/FacileStudio/gare/internal/storage"
)

func TestValidateTag(t *testing.T) {
	valid := []string{"api", "frontend-app", "client_123", "prod", "A1-B2"}
	for _, tag := range valid {
		if err := storage.ValidateTag(tag); err != nil {
			t.Errorf("expected valid tag: %s, got error: %v", tag, err)
		}
	}

	invalid := []string{"", "   ", "api/v1", "app.domain", "tag with spaces", "!invalid"}
	for _, tag := range invalid {
		if err := storage.ValidateTag(tag); err == nil {
			t.Errorf("expected invalid tag: %s, got nil error", tag)
		}
	}
}

func TestNormalizeTag(t *testing.T) {
	if got := storage.NormalizeTag("  PROD  "); got != "prod" {
		t.Errorf("got %q, want %q", got, "prod")
	}
	if got := storage.NormalizeTag("API-Service"); got != "api-service" {
		t.Errorf("got %q, want %q", got, "api-service")
	}
}

func TestAppConfigAddTags(t *testing.T) {
	cfg := &storage.AppConfig{Name: "testapp"}
	if err := cfg.AddTags("prod", "API", "  Frontend  "); err != nil {
		t.Fatalf("AddTags failed: %v", err)
	}
	if !cfg.HasTag("prod") || !cfg.HasTag("api") || !cfg.HasTag("frontend") {
		t.Errorf("missing expected tags: %v", cfg.Tags)
	}
	if cfg.HasTag("staging") {
		t.Errorf("unexpected tag staging found")
	}
	if err := cfg.AddTags("PROD"); err != nil {
		t.Fatalf("AddTags duplicate failed: %v", err)
	}
	if len(cfg.Tags) != 3 {
		t.Errorf("expected 3 tags, got %d", len(cfg.Tags))
	}
	if err := cfg.AddTags("invalid tag!"); err == nil {
		t.Errorf("expected error adding invalid tag")
	}
}

func TestAppConfigAddTagsAtomic(t *testing.T) {
	cfg := &storage.AppConfig{Name: "testapp", Tags: []string{"initial"}}
	err := cfg.AddTags("valid-1", "bad tag!", "valid-2")
	if err == nil {
		t.Fatal("expected error on invalid tag in batch")
	}
	if len(cfg.Tags) != 1 || cfg.Tags[0] != "initial" {
		t.Fatalf("expected tags to remain unchanged on batch error, got: %v", cfg.Tags)
	}
}

func TestAppConfigNilSafety(t *testing.T) {
	var nilCfg *storage.AppConfig
	if err := nilCfg.AddTags("tag"); err == nil {
		t.Error("expected error adding tags to nil config")
	}
	if nilCfg.RemoveTags("tag") {
		t.Error("expected false removing tags from nil config")
	}
	if nilCfg.HasTag("tag") {
		t.Error("expected false for HasTag on nil config")
	}
}

func TestAppConfigRemoveTags(t *testing.T) {
	cfg := &storage.AppConfig{Name: "testapp", Tags: []string{"prod", "api"}}
	if !cfg.RemoveTags("api") {
		t.Errorf("expected RemoveTags(api) to return true")
	}
	if cfg.HasTag("api") {
		t.Errorf("expected api to be removed")
	}
	if len(cfg.Tags) != 1 {
		t.Errorf("expected 1 tag remaining, got %d", len(cfg.Tags))
	}
	if cfg.RemoveTags("notfound") {
		t.Errorf("expected RemoveTags(notfound) to return false")
	}
}

func TestFilterAppsByTag(t *testing.T) {
	app1 := &storage.AppConfig{Name: "app1", Tags: []string{"client-a", "prod"}}
	app2 := &storage.AppConfig{Name: "app2", Tags: []string{"client-b", "prod"}}
	app3 := &storage.AppConfig{Name: "app3", Tags: []string{"client-a", "staging"}}
	apps := []*storage.AppConfig{app1, nil, app2, app3}

	filtered := storage.FilterAppsByTag(apps, "client-a")
	if len(filtered) != 2 {
		t.Errorf("expected 2 apps with client-a, got %d", len(filtered))
	}

	filteredProd := storage.FilterAppsByTag(apps, "prod")
	if len(filteredProd) != 2 {
		t.Errorf("expected 2 apps with prod, got %d", len(filteredProd))
	}

	filteredNone := storage.FilterAppsByTag(apps, "nonexistent")
	if len(filteredNone) != 0 {
		t.Errorf("expected 0 apps with nonexistent, got %d", len(filteredNone))
	}
}

func TestCollectAllTags(t *testing.T) {
	app1 := &storage.AppConfig{Name: "app1", Tags: []string{"client-a", "prod", "prod", "PROD"}}
	app2 := &storage.AppConfig{Name: "app2", Tags: []string{"client-b", "prod"}}
	app3 := &storage.AppConfig{Name: "app3", Tags: []string{"client-a", "staging"}}
	apps := []*storage.AppConfig{app1, nil, app2, app3}

	all := storage.FilterAppsByTag(apps, "")
	if len(all) != 4 {
		t.Errorf("expected 4 elements with empty filter, got %d", len(all))
	}

	counts := storage.CollectAllTags(apps)
	if counts["client-a"] != 2 || counts["prod"] != 2 || counts["client-b"] != 1 || counts["staging"] != 1 {
		t.Errorf("unexpected tag counts: %+v", counts)
	}
}
