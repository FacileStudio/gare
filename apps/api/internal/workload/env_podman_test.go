package workload

import (
	"maps"
	"os"
	"os/exec"
	"slices"
	"strings"
	"testing"

	"github.com/FacileStudio/gare/apps/api/internal/dotenv"
	"github.com/FacileStudio/gare/apps/api/internal/storage"
	"github.com/FacileStudio/gare/apps/api/internal/systemd"
)

// podmanEnvCheckImage is the image the env check runs printenv in, pulled once.
const podmanEnvCheckImage = "docker.io/library/alpine:3.20"

// TestStaticEnvReachesTheContainerExactly runs the static unit's env path for real: systemd parses
// the app env file (dotenv.Parse reads it the same way) into podman's environment, and podman reads
// the generated variable list. Feeding dotenv.Format's file to `--env-file` directly would keep the
// quotes and cut a multi-line value at its first line, which is why the list exists.
func TestStaticEnvReachesTheContainerExactly(t *testing.T) {
	skipWithoutPodmanWorkloads(t)
	app := testApp(t, &storage.AppConfig{Name: "site", AppType: "static"}, t.TempDir())
	want := map[string]string{
		"HASH": "ab #cd", "QUOTED": `"q"`, "LEAD": " lead", "MULTI": "l1\nl2",
		"DOLLAR": "$HOME `x`", "SINGLE": "it's", "BARE": "a=b",
	}
	if err := storage.SetAppEnv(app.Dir, want); err != nil {
		t.Fatal(err)
	}
	if err := writePodmanEnv(app); err != nil {
		t.Fatal(err)
	}
	keys := slices.Sorted(maps.Keys(want))
	got := containerEnv(t, storage.GetAppEnvPath(app.Dir), keys)
	for i, k := range keys {
		if value := strings.TrimSuffix(got[i], "\n"); value != want[k] {
			t.Errorf("%s reached the container as %q, want %q", k, value, want[k])
		}
	}
}

// containerEnv runs the check container the way the static unit does: the app env file parsed into
// podman's environment, its variable list as `--env-file`. It returns each key's printenv output.
func containerEnv(t *testing.T, envFile string, keys []string) []string {
	t.Helper()
	parsed, err := dotenv.LoadFile(envFile)
	if err != nil {
		t.Fatal(err)
	}
	script := `for k in "$@"; do printenv "$k"; printf '\0'; done`
	args := append([]string{"run", "--rm", "--env-file", systemd.PodmanEnvFile(envFile),
		podmanEnvCheckImage, "sh", "-c", script, "sh"}, keys...)
	cmd := exec.CommandContext(t.Context(), "podman", args...)
	cmd.Env = os.Environ()
	for k, v := range parsed {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("podman run: %v", err)
	}
	return strings.Split(string(out), "\x00")
}
