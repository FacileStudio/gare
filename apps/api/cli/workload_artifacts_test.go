package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/FacileStudio/gare/apps/api/internal/systemd"
	"github.com/FacileStudio/gare/apps/api/internal/testkit"
)

func TestAppUnitExists(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	exists, err := appUnitExists("never-deployed")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exists {
		t.Error("expected no artifacts for an undeployed app")
	}

	testkit.WriteUnitFile(t, "myapp", "[Unit]\nDescription="+systemd.KubeUnitDescription("myapp")+"\n")
	exists, err = appUnitExists("myapp")
	if err != nil || !exists {
		t.Errorf("expected a gare-written unit to count as deployed artifacts, got exists=%v err=%v", exists, err)
	}

	legacy := systemd.LegacyQuadletSources("older")[0]
	if err := os.MkdirAll(filepath.Dir(legacy), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(legacy, []byte("[Kube]\n"), 0644); err != nil {
		t.Fatal(err)
	}
	exists, err = appUnitExists("older")
	if err != nil || !exists {
		t.Errorf("an app deployed by an older gare must still be recognised, got exists=%v err=%v", exists, err)
	}
}
