package git

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func initTestGitRepo(t *testing.T) string {
	t.Helper()
	repoDir := t.TempDir()
	initCmd := "git init -b main && git config user.email test@example.com && git config user.name 'Test' && git config commit.gpgsign false && echo initial > README.md && git add README.md && git commit -m initial"
	cmd := exec.Command("sh", "-c", initCmd)
	cmd.Dir = repoDir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %v, out: %s", err, string(out))
	}
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
	if clonedHash, err := GetCommitHash(ctx, targetDir); err != nil || clonedHash != hash {
		t.Fatalf("hash mismatch: %v, got %q, want %q", err, clonedHash, hash)
	}
	if err := os.WriteFile(filepath.Join(sourceRepo, "README.md"), []byte("updated"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command("git", "-C", sourceRepo, "commit", "-am", "update").Run(); err != nil {
		t.Fatal(err)
	}
	if err := Pull(ctx, targetDir, Auth{}, &stdout, &stderr); err != nil {
		t.Fatalf("Pull failed: %v", err)
	}
	if content, err := os.ReadFile(filepath.Join(targetDir, "README.md")); err != nil || string(content) != "updated" {
		t.Errorf("unexpected content: %v, %q", err, string(content))
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
	branchDir := filepath.Join(t.TempDir(), "branch")
	var stdout, stderr bytes.Buffer
	bOpts := CloneOptions{RepoURL: sourceRepo, Branch: "main", TargetDir: branchDir, Stdout: &stdout, Stderr: &stderr}
	if err := Clone(ctx, bOpts); err != nil {
		t.Fatalf("Clone with branch failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(branchDir, "README.md")); err != nil {
		t.Fatalf("expected cloned file to exist: %v", err)
	}
}
