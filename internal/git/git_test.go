package git

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

func mustGit(t *testing.T, args ...string) {
	t.Helper()
	if out, err := exec.Command("git", args...).CombinedOutput(); err != nil {
		t.Fatalf("git %s failed: %v, out: %s", strings.Join(args, " "), err, out)
	}
}

func cloneForSync(t *testing.T) (string, string) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found in PATH")
	}
	sourceRepo := initTestGitRepo(t)
	targetDir := filepath.Join(t.TempDir(), "cloned")
	var out bytes.Buffer
	opts := CloneOptions{RepoURL: sourceRepo, Branch: "main", TargetDir: targetDir, Stdout: &out, Stderr: &out}
	if err := Clone(context.Background(), opts); err != nil {
		t.Fatalf("Clone failed: %v, out: %s", err, out.String())
	}
	return sourceRepo, targetDir
}

func verifyCloneAndSync(t *testing.T, ctx context.Context, sourceRepo, hash string) {
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
	mustGit(t, "-C", sourceRepo, "commit", "-am", "update")
	if err := os.WriteFile(filepath.Join(targetDir, "README.md"), []byte("local edit"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := Sync(ctx, SyncOptions{RepoDir: targetDir, Stdout: &stdout, Stderr: &stderr}); err != nil {
		t.Fatalf("Sync failed: %v, stderr: %s", err, stderr.String())
	}
	if content, err := os.ReadFile(filepath.Join(targetDir, "README.md")); err != nil || string(content) != "updated" {
		t.Errorf("a locally modified tracked file must be reset to the remote, got %v, %q", err, string(content))
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
	verifyCloneAndSync(t, ctx, sourceRepo, hash)
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

func TestSyncFollowsForcePush(t *testing.T) {
	ctx := context.Background()
	sourceRepo, targetDir := cloneForSync(t)
	if err := os.WriteFile(filepath.Join(sourceRepo, "README.md"), []byte("rewritten"), 0644); err != nil {
		t.Fatal(err)
	}
	mustGit(t, "-C", sourceRepo, "commit", "-a", "--amend", "-m", "rewritten")
	want, err := GetCommitHash(ctx, sourceRepo)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := Sync(ctx, SyncOptions{RepoDir: targetDir, Branch: "main", Stdout: &out, Stderr: &out}); err != nil {
		t.Fatalf("Sync after a force-push failed: %v, out: %s", err, out.String())
	}
	if got, err := GetCommitHash(ctx, targetDir); err != nil || got != want {
		t.Errorf("checkout must follow the rewritten history: %v, got %q, want %q", err, got, want)
	}
}

func TestSyncCleansUntrackedKeepsIgnored(t *testing.T) {
	sourceRepo, targetDir := cloneForSync(t)
	if err := os.WriteFile(filepath.Join(sourceRepo, ".gitignore"), []byte("build/\n"), 0644); err != nil {
		t.Fatal(err)
	}
	mustGit(t, "-C", sourceRepo, "add", ".gitignore")
	mustGit(t, "-C", sourceRepo, "commit", "-m", "ignore build")
	stray := filepath.Join(targetDir, "stray.txt")
	ignored := filepath.Join(targetDir, "build", "out.bin")
	for _, path := range []string{stray, ignored} {
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("x"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	var out bytes.Buffer
	if err := Sync(context.Background(), SyncOptions{RepoDir: targetDir, Branch: "main", Stdout: &out, Stderr: &out}); err != nil {
		t.Fatalf("Sync failed: %v, out: %s", err, out.String())
	}
	if _, err := os.Stat(stray); !os.IsNotExist(err) {
		t.Errorf("an untracked file must be removed, stat returned %v", err)
	}
	if _, err := os.Stat(ignored); err != nil {
		t.Errorf("an ignored file must survive the sync: %v", err)
	}
}

func TestSyncRejectsOptionBranch(t *testing.T) {
	_, targetDir := cloneForSync(t)
	for _, branch := range []string{"-x", "--upload-pack=touch pwned", "a..b"} {
		var out bytes.Buffer
		if err := Sync(context.Background(), SyncOptions{RepoDir: targetDir, Branch: branch, Stdout: &out, Stderr: &out}); err == nil {
			t.Errorf("branch %q must be rejected", branch)
		}
		if out.Len() != 0 {
			t.Errorf("branch %q must be rejected before git runs, got output %q", branch, out.String())
		}
	}
	opts := CloneOptions{RepoURL: targetDir, Branch: "-x", TargetDir: filepath.Join(t.TempDir(), "c")}
	if err := Clone(context.Background(), opts); err == nil {
		t.Error("Clone must reject a branch that starts with a dash")
	}
}
