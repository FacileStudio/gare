package storage

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"testing"
	"time"
)

// lockHolderEnv carries the store directory to the child process TestLockExcludesAnotherProcess
// starts, and marks that process as the lock holder.
const lockHolderEnv = "GARE_TEST_LOCK_HOLDER_DIR"

// lockAsync takes the lock on name in a goroutine and hands over its release function once held.
func lockAsync(t *testing.T, store *Store, name string) <-chan func() {
	t.Helper()
	acquired := make(chan func(), 1)
	go func() {
		unlock, err := store.Lock(context.Background(), name)
		if err != nil {
			t.Errorf("Lock(%q): %v", name, err)
			unlock = func() {}
		}
		acquired <- unlock
	}()
	return acquired
}

// assertLockFails proves name is held elsewhere: a lock attempt with a short deadline gives up.
func assertLockFails(t *testing.T, store *Store, name string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	if unlock, err := store.Lock(ctx, name); err == nil {
		unlock()
		t.Fatalf("Lock(%q) succeeded while another holder had it", name)
	}
}

func TestLockExcludesTheSameAppOnly(t *testing.T) {
	store := Open(t.TempDir())
	unlock, err := store.Lock(t.Context(), "a")
	if err != nil {
		t.Fatal(err)
	}
	waiter := lockAsync(t, store, "a")
	select {
	case <-waiter:
		t.Fatal("a second Lock(a) did not wait for the first to release")
	case <-time.After(200 * time.Millisecond):
	}
	other, err := store.Lock(t.Context(), "b")
	if err != nil {
		t.Fatalf("Lock(b) must not wait on a: %v", err)
	}
	other()
	cancelled, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := store.Lock(cancelled, "a"); err == nil {
		t.Fatal("Lock with a cancelled context must return an error while a is held")
	}
	unlock()
	select {
	case release := <-waiter:
		release()
	case <-time.After(5 * time.Second):
		t.Fatal("the waiting Lock(a) never acquired after the release")
	}
}

func TestLockSurvivesDeletingTheApp(t *testing.T) {
	store := Open(t.TempDir())
	if err := store.Save(&AppConfig{Name: "a"}); err != nil {
		t.Fatal(err)
	}
	unlock, err := store.Lock(t.Context(), "a")
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()
	if err := store.Delete("a"); err != nil {
		t.Fatal(err)
	}
	assertLockFails(t, store, "a")
	if apps, err := store.ListStrict(); err != nil || len(apps) != 0 {
		t.Fatalf("the lock directory must not count as an app: %v, %v", apps, err)
	}
}

func TestLockHolderProcess(t *testing.T) {
	dir := os.Getenv(lockHolderEnv)
	if dir == "" {
		t.Skip("runs only as the child process of TestLockExcludesAnotherProcess")
	}
	unlock, err := Open(dir).Lock(context.Background(), "a")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("locked")
	io.Copy(io.Discard, os.Stdin)
	unlock()
}

func TestLockExcludesAnotherProcess(t *testing.T) {
	dir := t.TempDir()
	child := exec.Command(os.Args[0], "-test.run=^TestLockHolderProcess$")
	child.Env = append(os.Environ(), lockHolderEnv+"="+dir)
	release, err := child.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	out, err := child.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := child.Start(); err != nil {
		t.Fatal(err)
	}
	if line, _ := bufio.NewReader(out).ReadString('\n'); line != "locked\n" {
		t.Fatalf("the child process did not take the lock: %q", line)
	}
	store := Open(dir)
	assertLockFails(t, store, "a")
	waiter := lockAsync(t, store, "a")
	release.Close()
	select {
	case unlock := <-waiter:
		unlock()
	case <-time.After(10 * time.Second):
		t.Fatal("the lock was not released when the child process let go")
	}
	child.Wait()
}
