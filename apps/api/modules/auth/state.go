package auth

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/FacileStudio/gare/apps/api/internal/atomicfile"
)

// fileState is porte/directory's StateStore over one file: the cursor and the
// erased subjects of the deletion receiver. It is a file rather than a row
// because gare keeps its own state on disk and Postgres holds porte's tables
// only.
type fileState struct {
	path string
}

// Load returns the saved state, or nothing before the first save, which is
// what porte/directory expects from a fresh install.
func (s fileState) Load(context.Context) ([]byte, error) {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	return data, err
}

// Save replaces the state atomically, so a crash mid-write never leaves a
// half file that would forget an erased person.
func (s fileState) Save(_ context.Context, state []byte) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	return atomicfile.WriteFile(s.path, state, 0o600)
}
