package atomicfile

import (
	"errors"
	"os"
	"path/filepath"
)

// WriteFile writes data to a temporary file, syncs it, renames it atomically to path, and syncs the parent directory.
func WriteFile(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	tmpName, err := writeTemp(dir, data, perm)
	if err != nil {
		return err
	}

	if err := os.Rename(tmpName, path); err != nil {
		return errors.Join(err, os.Remove(tmpName))
	}

	return syncDirectory(dir)
}

func writeTemp(dir string, data []byte, perm os.FileMode) (string, error) {
	tmp, err := os.CreateTemp(dir, ".atomic-*")
	if err != nil {
		return "", err
	}
	name := tmp.Name()
	err = writeAndSync(tmp, data, perm)
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return "", errors.Join(err, os.Remove(name))
	}
	return name, nil
}

func writeAndSync(tmp *os.File, data []byte, perm os.FileMode) error {
	if err := tmp.Chmod(perm); err != nil {
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		return err
	}
	return tmp.Sync()
}

func syncDirectory(dir string) error {
	dirFD, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer dirFD.Close()
	return dirFD.Sync()
}
