package workload

import (
	"testing"

	"github.com/FacileStudio/gare/apps/api/internal/storage"
)

// TestForSelectsTheAdapterAndItsEnvStore pins the one place a workload type is read: each type gets
// its own adapter, the env store follows it, and a type gare does not know is refused rather than
// run as a container.
func TestForSelectsTheAdapterAndItsEnvStore(t *testing.T) {
	cases := []struct {
		appType string
		want    Workload
		label   string
	}{
		{"", container{}, "manifest"},
		{"container", container{}, "manifest"},
		{"Static", static{}, "app env file"},
		{"compose", composeStack{}, "app env file"},
	}
	for _, tc := range cases {
		got, err := For(&storage.AppConfig{AppType: tc.appType})
		if err != nil {
			t.Fatalf("For(%q): %v", tc.appType, err)
		}
		if got != tc.want {
			t.Errorf("For(%q) = %T, want %T", tc.appType, got, tc.want)
		}
		if label := got.Env().Label; label != tc.label {
			t.Errorf("For(%q).Env() = %q, want %q", tc.appType, label, tc.label)
		}
	}
	if _, err := For(&storage.AppConfig{AppType: "kubernetes"}); err == nil {
		t.Error("an unknown workload type must be refused, not run as a container")
	}
	if _, err := For(nil); err == nil {
		t.Error("a missing configuration must be refused")
	}
}
