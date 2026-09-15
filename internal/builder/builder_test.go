package builder

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
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

func runGitCmd(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v, out: %s", args, err, string(out))
	}
}

func initTestGitRepo(t *testing.T) string {
	t.Helper()
	repoDir := t.TempDir()
	runGitCmd(t, "", "init", repoDir)
	runGitCmd(t, repoDir, "config", "user.email", "test@example.com")
	runGitCmd(t, repoDir, "config", "user.name", "Test User")
	runGitCmd(t, repoDir, "config", "commit.gpgsign", "false")
	readme := filepath.Join(repoDir, "README.md")
	if err := os.WriteFile(readme, []byte("initial"), 0644); err != nil {
		t.Fatal(err)
	}
	runGitCmd(t, repoDir, "add", "README.md")
	runGitCmd(t, repoDir, "commit", "-m", "initial commit")
	return repoDir
}

func verifyCloneAndPull(t *testing.T, ctx context.Context, sourceRepo, hash string) {
	t.Helper()
	targetDir := filepath.Join(t.TempDir(), "cloned")
	var stdout, stderr bytes.Buffer
	opts := CloneOptions{RepoURL: sourceRepo, TargetDir: targetDir, Stdout: &stdout, Stderr: &stderr}
	if err := Clone(ctx, opts); err != nil {
		t.Fatalf("Clone failed: %v, stderr: %s", err, stderr.String())
	}
	clonedHash, err := GetCommitHash(ctx, targetDir)
	if err != nil || clonedHash != hash {
		t.Fatalf("hash mismatch or error: %v, got %q, want %q", err, clonedHash, hash)
	}
	readme := filepath.Join(sourceRepo, "README.md")
	if err := os.WriteFile(readme, []byte("updated"), 0644); err != nil {
		t.Fatal(err)
	}
	runGitCmd(t, sourceRepo, "commit", "-am", "update commit")
	stdout.Reset()
	stderr.Reset()
	if err := Pull(ctx, targetDir, &stdout, &stderr); err != nil {
		t.Fatalf("Pull failed: %v, stderr: %s", err, stderr.String())
	}
	content, err := os.ReadFile(filepath.Join(targetDir, "README.md"))
	if err != nil || string(content) != "updated" {
		t.Errorf("readfile error or unexpected content: %v, %q", err, string(content))
	}
}

func verifyCloneBranch(t *testing.T, ctx context.Context, sourceRepo string) {
	t.Helper()
	targetBranchDir := filepath.Join(t.TempDir(), "cloned_branch")
	branchCmd := exec.Command("git", "-C", sourceRepo, "rev-parse", "--abbrev-ref", "HEAD")
	branchOut, err := branchCmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	branch := string(bytes.TrimSpace(branchOut))
	var stdout, stderr bytes.Buffer
	opts := CloneOptions{RepoURL: sourceRepo, Branch: branch, TargetDir: targetBranchDir, Stdout: &stdout, Stderr: &stderr}
	if err := Clone(ctx, opts); err != nil {
		t.Fatalf("Clone with branch failed: %v, stderr: %s", err, stderr.String())
	}
	if _, err := os.Stat(filepath.Join(targetBranchDir, "README.md")); err != nil {
		t.Fatalf("expected cloned file to exist: %v", err)
	}
}

func TestGitOperations(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found in PATH")
	}
	ctx := context.Background()
	sourceRepo := initTestGitRepo(t)
	hash, err := GetCommitHash(ctx, sourceRepo)
	if err != nil || len(hash) == 0 {
		t.Fatalf("GetCommitHash failed: %v", err)
	}
	verifyCloneAndPull(t, ctx, sourceRepo, hash)
	verifyCloneBranch(t, ctx, sourceRepo)
}

func TestPodmanOperations(t *testing.T) {
	if _, err := exec.LookPath("podman"); err != nil {
		t.Skip("podman not found in PATH")
	}

	ctx := context.Background()
	repoDir := t.TempDir()
	cfPath := filepath.Join(repoDir, "Containerfile")
	if err := os.WriteFile(cfPath, []byte("FROM scratch\n"), 0644); err != nil {
		t.Fatal(err)
	}

	imageName := "localhost/gare-builder-test:scratch"
	var stdout, stderr bytes.Buffer
	if err := Build(ctx, repoDir, imageName, &stdout, &stderr); err != nil {
		t.Fatalf("Build failed: %v, stderr: %s", err, stderr.String())
	}

	if err := RemoveImage(ctx, imageName); err != nil {
		t.Fatalf("RemoveImage failed: %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	if err := PruneImages(ctx, &stdout, &stderr); err != nil {
		t.Fatalf("PruneImages failed: %v, stderr: %s", err, stderr.String())
	}
}
