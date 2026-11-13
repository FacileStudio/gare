package deploys

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

// maxIDAttempts bounds the search for a free id when deploys of one app start in the same
// millisecond.
const maxIDAttempts = 1000

// Request is what a caller knows about a deploy when it creates its record: who started it, the
// commit when it is already known, and the deploy a rollback restores.
type Request struct {
	Trigger      Trigger
	Commit       string
	RolledBackTo string
}

// Start creates a running deploy's record and empty log, and returns the recorder that writes them
// until the deploy finishes.
func (h History) Start(req Request) (*Recorder, error) {
	rec, err := h.create(req, StatusRunning)
	if err != nil {
		return nil, err
	}
	return h.recorder(rec)
}

// Queue creates the record of a deploy that waits for its turn, so a client sees it in the history
// before it runs. Resume later turns it into a running deploy.
func (h History) Queue(req Request) (Deploy, error) {
	return h.create(req, StatusQueued)
}

// Resume marks a queued deploy as running and returns the recorder that writes it from then on.
func (h History) Resume(id string) (*Recorder, error) {
	rec, err := h.Get(id)
	if err != nil {
		return nil, err
	}
	if rec.Status != StatusQueued {
		return nil, fmt.Errorf("deploy %s is %s, not queued", id, rec.Status)
	}
	rec.Status = StatusRunning
	rec.StartedAt = time.Now().UTC()
	r, err := h.recorder(rec)
	if err != nil {
		return nil, err
	}
	return r, r.save()
}

// create makes a new record directory with its record and an empty log, then prunes the records
// beyond Keep.
func (h History) create(req Request, status Status) (Deploy, error) {
	now := time.Now().UTC()
	id, err := h.makeDir(now, req.Commit)
	if err != nil {
		return Deploy{}, err
	}
	rec := Deploy{ID: id, App: h.app, Commit: req.Commit, Trigger: req.Trigger, Status: status, StartedAt: now}
	if req.RolledBackTo != "" {
		rec.RolledBackTo = &req.RolledBackTo
	}
	if err := os.WriteFile(filepath.Join(h.dir, id, logFile), nil, 0600); err != nil {
		return Deploy{}, err
	}
	if err := writeRecord(filepath.Join(h.dir, id), rec); err != nil {
		return Deploy{}, err
	}
	return rec, h.prune()
}

// makeDir creates the directory of a new record under an id no other record holds, moving the id
// forward a millisecond at a time while it collides.
func (h History) makeDir(at time.Time, commit string) (string, error) {
	if err := os.MkdirAll(h.dir, 0700); err != nil {
		return "", fmt.Errorf("failed to create the deploy history: %w", err)
	}
	for range maxIDAttempts {
		id := newID(at, commit)
		err := os.Mkdir(filepath.Join(h.dir, id), 0700)
		if err == nil {
			return id, nil
		}
		if !errors.Is(err, fs.ErrExist) {
			return "", fmt.Errorf("failed to create deploy record %s: %w", id, err)
		}
		at = at.Add(time.Millisecond)
	}
	return "", fmt.Errorf("no free deploy id for %s", h.app)
}

// recorder opens a record's log for appending and returns the recorder that owns it.
func (h History) recorder(rec Deploy) (*Recorder, error) {
	r := &Recorder{dir: filepath.Join(h.dir, rec.ID), rec: rec, atLineStart: true}
	var err error
	r.log, err = os.OpenFile(filepath.Join(r.dir, logFile), os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0600)
	if err != nil {
		return nil, fmt.Errorf("failed to open the log of deploy %s: %w", rec.ID, err)
	}
	return r, nil
}
