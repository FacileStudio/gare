package storage

import (
	"os"
	"testing"
)

func TestSetAppEnvPreservesOtherValues(t *testing.T) {
	appDir := t.TempDir()
	values := map[string]string{"HASH": "ab #cd", "QUOTED": `"q"`, "LEAD": " lead", "MULTI": "l1\nl2"}
	if err := SetAppEnv(appDir, values); err != nil {
		t.Fatalf("SetAppEnv: %v", err)
	}
	if err := SetAppEnv(appDir, map[string]string{"OTHER": "1"}); err != nil {
		t.Fatalf("second SetAppEnv: %v", err)
	}
	if err := UnsetAppEnv(appDir, []string{"OTHER"}); err != nil {
		t.Fatalf("UnsetAppEnv: %v", err)
	}
	got, err := GetAppEnv(appDir)
	if err != nil {
		t.Fatalf("GetAppEnv: %v", err)
	}
	for k, v := range values {
		if got[k] != v {
			t.Errorf("%s: set %q, read back %q", k, v, got[k])
		}
	}
}

func TestAppEnvFileIsOwnerOnly(t *testing.T) {
	appDir := t.TempDir()
	if err := EnsureAppEnvFile(appDir); err != nil {
		t.Fatalf("EnsureAppEnvFile: %v", err)
	}
	if err := SetAppEnv(appDir, map[string]string{"K": "v"}); err != nil {
		t.Fatalf("SetAppEnv: %v", err)
	}
	info, err := os.Stat(GetAppEnvPath(appDir))
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("env file mode %v, want 0600", info.Mode().Perm())
	}
}
