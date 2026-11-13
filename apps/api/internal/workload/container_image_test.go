package workload

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/FacileStudio/gare/apps/api/internal/storage"
	"github.com/FacileStudio/gare/apps/api/internal/testkit"
)

// manifestText returns the manifest gare keeps for the app stored in appDir.
func manifestText(t *testing.T, appDir string) string {
	t.Helper()
	data, err := os.ReadFile(storage.GetManifestPath(appDir))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestDeployManifestRunsTheCommitTagAndRollbackRepointsIt(t *testing.T) {
	podmanLog := testkit.StubPodman(t)
	appDir := filepath.Join(t.TempDir(), "app")
	if err := os.MkdirAll(filepath.Join(appDir, "repo"), 0755); err != nil {
		t.Fatal(err)
	}
	app := testApp(t, &storage.AppConfig{Name: "web", Port: 8100, ContainerPort: 3000}, appDir)
	if err := syncManifest(app, "bbbbbbb"); err != nil {
		t.Fatal(err)
	}
	if got := manifestText(t, appDir); !strings.Contains(got, "image: localhost/web:bbbbbbb") {
		t.Fatalf("a deploy must run the image of its commit, got:\n%s", got)
	}
	if err := PointImage(t.Context(), app, "aaaaaaa"); err != nil {
		t.Fatal(err)
	}
	if got := manifestText(t, appDir); !strings.Contains(got, "image: localhost/web:aaaaaaa") {
		t.Errorf("a rollback must point the manifest at the earlier tag, got:\n%s", got)
	}
	want := "tag localhost/web:aaaaaaa localhost/web:latest"
	if calls := testkit.RecordedCalls(t, podmanLog); !slices.Contains(calls, want) {
		t.Errorf("a rollback must move :latest to the restored image, podman calls %v", calls)
	}
}

func TestCanRollBackOnlyContainers(t *testing.T) {
	cases := map[string]bool{"": true, "container": true, "static": false, "compose": false}
	for appType, want := range cases {
		if got := CanRollBack(&storage.AppConfig{Name: "a", AppType: appType}); got != want {
			t.Errorf("CanRollBack(%q) = %v, want %v", appType, got, want)
		}
	}
}
