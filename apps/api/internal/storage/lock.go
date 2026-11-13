package storage

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/FacileStudio/gare/apps/api/internal/appname"
)

// lockPollInterval is how long Lock sleeps between two attempts on a held lock. flock has no
// timeout of its own, so Lock polls a non-blocking attempt to let a cancelled context end the wait.
const lockPollInterval = 50 * time.Millisecond

// lockDir names the directory under the store's base that holds one lock file per app. The locks
// live outside the app directories so that destroy, which holds an app's lock while it deletes that
// app's directory, still holds a lock on a file that exists. A lock file is never removed: unlinking
// it while another process waits on it would let a third process lock a fresh file of the same name.
const lockDir = ".locks"

// Lock takes the exclusive lock on a named app and returns the function that releases it. The lock
// is an flock on a file under the store, so it excludes every gare process on the host, the CLI and
// the webhook daemon alike, and the kernel drops it when its holder exits. Lock waits while another
// holder has it, and gives up with the context's error once ctx is done.
func (s *Store) Lock(ctx context.Context, name string) (func(), error) {
	if err := appname.Validate(name); err != nil {
		return nil, err
	}
	dir := filepath.Join(s.baseDir, lockDir)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create the lock directory: %w", err)
	}
	file, err := os.OpenFile(filepath.Join(dir, name+".lock"), os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return nil, fmt.Errorf("failed to open the lock for app %q: %w", name, err)
	}
	if err := waitForFlock(ctx, file); err != nil {
		return nil, errors.Join(fmt.Errorf("failed to lock app %q: %w", name, err), file.Close())
	}
	return func() { file.Close() }, nil
}

// waitForFlock retries a non-blocking exclusive flock on file until it succeeds, fails for a reason
// other than a competing holder, or ctx is done.
func waitForFlock(ctx context.Context, file *os.File) error {
	for {
		err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if !errors.Is(err, syscall.EWOULDBLOCK) {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(lockPollInterval):
		}
	}
}
