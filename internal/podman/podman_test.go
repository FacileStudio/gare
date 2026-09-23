package podman

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func checkDetect(t *testing.T, dir, want string, wantErr bool) {
	t.Helper()
	got, err := DetectContainerfile(dir)
	if (err != nil) != wantErr || got != want {
		t.Fatalf("got %q, %v; want %q (wantErr=%v)", got, err, want, wantErr)
	}
}

func TestDetectContainerfile(t *testing.T) {
	tmpDir := t.TempDir()
	checkDetect(t, tmpDir, "", true)
	cf := filepath.Join(tmpDir, "Containerfile")
	df := filepath.Join(tmpDir, "Dockerfile")
	if err := os.WriteFile(cf, []byte("FROM scratch\n"), 0644); err != nil {
		t.Fatal(err)
	}
	checkDetect(t, tmpDir, "Containerfile", false)
	if err := os.WriteFile(df, []byte("FROM scratch\n"), 0644); err != nil {
		t.Fatal(err)
	}
	checkDetect(t, tmpDir, "Containerfile", false)
	if err := os.Remove(cf); err != nil {
		t.Fatal(err)
	}
	checkDetect(t, tmpDir, "Dockerfile", false)
	if err := os.Remove(df); err != nil || os.Mkdir(cf, 0755) != nil {
		t.Fatal("failed removing df or making cf dir")
	}
	checkDetect(t, tmpDir, "", true)
}

func buildAndRemove(t *testing.T, ctx context.Context, opts BuildOptions) {
	t.Helper()
	if err := Build(ctx, opts); err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	if err := RemoveImage(ctx, opts.ImageName); err != nil {
		t.Fatalf("RemoveImage failed: %v", err)
	}
}

func TestPodmanOperations(t *testing.T) {
	if _, err := exec.LookPath("podman"); err != nil {
		t.Skip("podman not found in PATH")
	}
	ctx := context.Background()
	repoDir := t.TempDir()
	cf := filepath.Join(repoDir, "Containerfile")
	customCf := filepath.Join(repoDir, "custom.Containerfile")
	content := []byte("FROM scratch\n")
	if err := os.WriteFile(cf, content, 0644); err != nil || os.WriteFile(customCf, content, 0644) != nil {
		t.Fatal("failed creating containerfiles")
	}
	var stdout, stderr bytes.Buffer
	buildAndRemove(t, ctx, BuildOptions{RepoDir: repoDir, ImageName: "localhost/gare-test:default", Stdout: &stdout, Stderr: &stderr})
	customOpts := BuildOptions{
		RepoDir:       repoDir,
		ImageName:     "localhost/gare-test:custom",
		Containerfile: "custom.Containerfile",
		ContextDir:    ".",
		Stdout:        &stdout,
		Stderr:        &stderr,
	}
	buildAndRemove(t, ctx, customOpts)
	stdout.Reset()
	stderr.Reset()
	if err := PruneImages(ctx, &stdout, &stderr); err != nil {
		t.Fatalf("PruneImages failed: %v", err)
	}
}

func TestRunBuildCommand(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	var stdout, stderr bytes.Buffer
	if err := RunBuildCommand(ctx, dir, "echo hello", &stdout, &stderr); err != nil {
		t.Fatalf("RunBuildCommand failed: %v", err)
	}
	if strings.TrimSpace(stdout.String()) != "hello" {
		t.Fatalf("got %q, want %q", stdout.String(), "hello")
	}
	if err := RunBuildCommand(ctx, dir, "exit 1", &stdout, &stderr); err == nil {
		t.Fatal("expected error from failing build command")
	}
}
