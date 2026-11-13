package manifest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSetImagePointsOnlyTheAppRepository(t *testing.T) {
	path := filepath.Join(t.TempDir(), "manifest.yaml")
	content := "apiVersion: v1\nkind: Pod\nspec:\n  containers:\n" +
		"  - name: app\n    image: localhost/app:latest\n" +
		"  - name: bare\n    image: localhost/app\n" +
		"  - name: side\n    image: localhost:5000/app:latest\n" +
		"  - name: other\n    image: localhost/other:latest\n"
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	if err := SetImage(path, "localhost/app", "localhost/app:abc1234"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	if strings.Count(got, "image: localhost/app:abc1234") != 2 {
		t.Errorf("both references to the app image must be re-pointed:\n%s", got)
	}
	for _, kept := range []string{"localhost:5000/app:latest", "localhost/other:latest"} {
		if !strings.Contains(got, kept) {
			t.Errorf("image %s must be left alone:\n%s", kept, got)
		}
	}
}
