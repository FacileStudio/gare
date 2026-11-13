package deploys

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sync"

	"github.com/FacileStudio/gare/apps/api/internal/atomicfile"
)

// Recorder is the single writer of one deploy's record and log. Its methods are safe to call from
// several goroutines, since a subprocess streams its stdout and stderr on two of them.
type Recorder struct {
	dir         string
	mu          sync.Mutex
	rec         Deploy
	log         *os.File
	atLineStart bool
	err         error
}

// Write appends subprocess output to the log as it arrives. It never fails: the recorder sits next
// to the terminal in a multi-writer, and a full disk must not cut the operator's own output short.
// The first error is kept and reported when the deploy finishes.
func (r *Recorder) Write(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.append(p)
	return len(p), nil
}

// Line appends one line to the log, starting it on a line of its own when subprocess output left
// the previous one unterminated.
func (r *Recorder) Line(text string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.atLineStart {
		r.append([]byte("\n"))
	}
	r.append([]byte(text + "\n"))
}

// SetCommit records the commit the deploy runs once the checkout is synced.
func (r *Recorder) SetCommit(commit string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rec.Commit = commit
	return r.save()
}

// Record returns the deploy as recorded so far.
func (r *Recorder) Record() Deploy {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.rec
}

// append writes to the log and remembers whether it ended a line. The caller holds the lock.
func (r *Recorder) append(p []byte) {
	if len(p) == 0 {
		return
	}
	if _, err := r.log.Write(p); err != nil && r.err == nil {
		r.err = err
	}
	r.atLineStart = p[len(p)-1] == '\n'
}

// save writes the record, unless its directory is gone: a destroy that ran while the deploy waited
// removed the app, and writing would bring a stray directory back. The caller holds the lock.
func (r *Recorder) save() error {
	if _, err := os.Stat(r.dir); errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return writeRecord(r.dir, r.rec)
}

// writeRecord replaces a record atomically, so a reader never sees half of one.
func writeRecord(dir string, rec Deploy) error {
	data, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return err
	}
	return atomicfile.WriteFile(filepath.Join(dir, recordFile), append(data, '\n'), 0600)
}
