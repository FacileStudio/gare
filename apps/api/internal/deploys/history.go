package deploys

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
)

// Keep is how many deploy records an app keeps; older ones are deleted when a new one starts.
const Keep = 50

const (
	recordFile = "record.json"
	logFile    = "log.txt"
)

// ErrNotFound reports a deploy id the app has no record of.
var ErrNotFound = errors.New("deploy not found")

// History is one app's deploy records, kept under apps/<name>/deploys/.
type History struct {
	dir string
	app string
}

// Open returns the history of the app named app, whose storage directory is appDir.
func Open(appDir, app string) History {
	return History{dir: filepath.Join(appDir, "deploys"), app: app}
}

// List returns every deploy the app kept, newest first. An app that never deployed has none.
func (h History) List() ([]Deploy, error) {
	ids, err := h.ids()
	if err != nil {
		return nil, err
	}
	records := make([]Deploy, 0, len(ids))
	for _, id := range ids {
		rec, err := h.Get(id)
		if errors.Is(err, ErrNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}
		records = append(records, rec)
	}
	return records, nil
}

// Get returns one deploy's record.
func (h History) Get(id string) (Deploy, error) {
	if err := ValidateID(id); err != nil {
		return Deploy{}, err
	}
	data, err := os.ReadFile(filepath.Join(h.dir, id, recordFile))
	if errors.Is(err, fs.ErrNotExist) {
		return Deploy{}, fmt.Errorf("%w: %s", ErrNotFound, id)
	}
	if err != nil {
		return Deploy{}, err
	}
	var rec Deploy
	if err := json.Unmarshal(data, &rec); err != nil {
		return Deploy{}, fmt.Errorf("failed to read deploy record %s: %w", id, err)
	}
	return rec, nil
}

// OpenLog opens one deploy's log for reading. A running deploy's log keeps growing behind it.
func (h History) OpenLog(id string) (*os.File, error) {
	if err := ValidateID(id); err != nil {
		return nil, err
	}
	path := filepath.Join(h.dir, id, logFile)
	if _, err := os.Stat(path); errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("%w: %s", ErrNotFound, id)
	}
	return os.Open(path)
}

// ids returns the id of every record directory, newest first, ignoring anything else in the
// history directory.
func (h History) ids() ([]string, error) {
	entries, err := os.ReadDir(h.dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var ids []string
	for _, entry := range entries {
		if entry.IsDir() && ValidateID(entry.Name()) == nil {
			ids = append(ids, entry.Name())
		}
	}
	slices.Sort(ids)
	slices.Reverse(ids)
	return ids, nil
}

// prune deletes the records beyond the newest Keep, so the history of an app that deploys on every
// push does not grow without bound.
func (h History) prune() error {
	ids, err := h.ids()
	if err != nil || len(ids) <= Keep {
		return err
	}
	var errs []error
	for _, id := range ids[Keep:] {
		errs = append(errs, os.RemoveAll(filepath.Join(h.dir, id)))
	}
	return errors.Join(errs...)
}
