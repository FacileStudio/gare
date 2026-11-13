package storage

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

// ErrUnreadableApps reports app directories whose configuration could not be read.
var ErrUnreadableApps = errors.New("unreadable app configuration")

// ListApps returns every readable application sorted by name. It skips an app directory whose
// configuration cannot be read, so one broken app does not hide the others; callers that must not
// overlook an app use ListAppsStrict.
func ListApps(baseDir string) ([]*AppConfig, error) {
	apps, _, err := scanApps(baseDir)
	return apps, err
}

// ListAppsStrict returns every application, or an error wrapping ErrUnreadableApps that names each
// app directory whose configuration could not be read.
func ListAppsStrict(baseDir string) ([]*AppConfig, error) {
	apps, broken, err := scanApps(baseDir)
	if err != nil {
		return nil, err
	}
	if len(broken) > 0 {
		return nil, fmt.Errorf("%w: %v", ErrUnreadableApps, broken)
	}
	return apps, nil
}

// UnreadableApps names the app directories under baseDir whose configuration exists but cannot be
// read or parsed. A directory with no configuration yet is not an app and is not reported.
func UnreadableApps(baseDir string) ([]string, error) {
	_, broken, err := scanApps(baseDir)
	return broken, err
}

func scanApps(baseDir string) ([]*AppConfig, []string, error) {
	if baseDir == "" {
		baseDir = DefaultBaseDir()
	}
	entries, err := os.ReadDir(baseDir)
	if os.IsNotExist(err) {
		return []*AppConfig{}, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	apps := []*AppConfig{}
	var broken []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		cfg, err := LoadConfig(filepath.Join(baseDir, entry.Name()))
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
