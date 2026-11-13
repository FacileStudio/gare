package storage

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"sort"
	"strings"
)

// ErrUnreadableApps reports app directories whose configuration could not be read.
var ErrUnreadableApps = errors.New("unreadable app configuration")

// List returns every readable application sorted by name. It skips an app directory whose
// configuration cannot be read, so one broken app does not hide the others; callers that must not
// overlook an app use ListStrict.
func (s *Store) List() ([]*AppConfig, error) {
	apps, _, err := s.scan()
	return apps, err
}

// ListStrict returns every application, or an error wrapping ErrUnreadableApps that names each
// app directory whose configuration could not be read.
func (s *Store) ListStrict() ([]*AppConfig, error) {
	apps, broken, err := s.scan()
	if err != nil {
		return nil, err
	}
	if len(broken) > 0 {
		return nil, fmt.Errorf("%w: %v", ErrUnreadableApps, broken)
	}
	return apps, nil
}

// Unreadable names the app directories whose configuration exists but cannot be read or parsed. A
// directory with no configuration yet is not an app and is not reported.
func (s *Store) Unreadable() ([]string, error) {
	_, broken, err := s.scan()
	return broken, err
}

// scan reads every app directory once and sorts its findings into readable apps and broken ones.
// Hidden entries are skipped: no app name starts with a dot, and the lock directory does.
func (s *Store) scan() ([]*AppConfig, []string, error) {
	entries, err := os.ReadDir(s.baseDir)
	if os.IsNotExist(err) {
		return []*AppConfig{}, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	apps := []*AppConfig{}
	var broken []string
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		cfg, err := loadConfig(s.AppDir(entry.Name()))
		switch {
		case errors.Is(err, fs.ErrNotExist):
		case err != nil:
			broken = append(broken, entry.Name())
		default:
			apps = append(apps, cfg)
		}
	}
	sort.Slice(apps, func(i, j int) bool { return apps[i].Name < apps[j].Name })
	return apps, broken, nil
}
