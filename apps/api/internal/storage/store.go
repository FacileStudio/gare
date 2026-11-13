package storage

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/FacileStudio/gare/apps/api/internal/appname"
	"github.com/FacileStudio/gare/apps/api/internal/xdg"
)

// Store is the directory every managed application lives under. It is opened on one base directory
// so the CLI, the webhook daemon and the tests agree on where an app is kept without each of them
// resolving the XDG default on its own.
type Store struct {
	baseDir string
}

// Open returns the store kept under baseDir.
func Open(baseDir string) *Store {
	return &Store{baseDir: baseDir}
}

// Default returns the store under the XDG data directory of the current user, the one the CLI and
// the daemon share on a host.
func Default() *Store {
	return Open(DefaultBaseDir())
}

// DefaultBaseDir returns the default directory path for app storage.
func DefaultBaseDir() string {
	return filepath.Join(xdg.DataHome(), "gare", "apps")
}

// AppDir returns the directory a named app's files live in.
func (s *Store) AppDir(name string) string {
	return filepath.Join(s.baseDir, name)
}

// Load reads a named app's configuration.
func (s *Store) Load(name string) (*AppConfig, error) {
	return loadConfig(s.AppDir(name))
}

// Save writes an app's configuration under the directory its name selects. It refuses a name that
// is not a valid app name, because an empty or dotted name would land the file outside the app's own
// directory.
func (s *Store) Save(cfg *AppConfig) error {
	if err := appname.Validate(cfg.Name); err != nil {
		return err
	}
	return saveConfig(s.AppDir(cfg.Name), cfg)
}

// Delete removes a named app's directory and everything in it. It refuses an invalid name, since an
// empty one would remove the whole store. The app's lock file lives outside that directory, so a
// caller that holds the lock while it deletes keeps holding it.
func (s *Store) Delete(name string) error {
	if err := appname.Validate(name); err != nil {
		return fmt.Errorf("refusing to delete app storage: %w", err)
	}
	return os.RemoveAll(s.AppDir(name))
}
