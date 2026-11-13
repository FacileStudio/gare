package engine

import (
	"context"
	"strings"
	"testing"

	"github.com/FacileStudio/gare/apps/api/internal/storage"
)

// TestAddDomainRejectsAHostAnotherAppServesOnItsDefaultPort pins uniqueness by host: Caddy reads
// example.com as example.com:443, so a second app cannot claim it by spelling the port out.
func TestAddDomainRejectsAHostAnotherAppServesOnItsDefaultPort(t *testing.T) {
	store := storage.Open(t.TempDir())
	for _, cfg := range []*storage.AppConfig{
		{Name: "owner", Port: 8000, Domains: []string{"example.com"}},
		{Name: "intruder", Port: 8001},
	} {
		if err := store.Save(cfg); err != nil {
			t.Fatal(err)
		}
	}
	eng, _ := newTestEngine(store, Settings{})

	err := eng.AddDomain(context.Background(), "intruder", "Example.com:443")
	if err == nil || !strings.Contains(err.Error(), `"owner"`) {
		t.Fatalf("expected the domain to be refused as owned by owner, got: %v", err)
	}
}
