package atomicfile

import (
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
		if rmErr := os.Remove(tmpName); rmErr != nil {
			return err
		}
		return err
	}

	return syncDirectory(dir)
}

func writeTemp(dir string, data []byte, perm os.FileMode) (string, error) {
	tmp, err := os.CreateTemp(dir, ".atomic-*")
	if err != nil {
		return "", err
	}
	name := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		cleanTemp(tmp, name)
		return "", err
	}
	if err := tmp.Sync(); err != nil {
		cleanTemp(tmp, name)
		return "", err
	}
	if err := tmp.Close(); err != nil {
		if rmErr := os.Remove(name); rmErr != nil {
			return "", err
		}
		return "", err
	}
	if err := os.Chmod(name, perm); err != nil {
		if rmErr := os.Remove(name); rmErr != nil {
			return "", err
		}
		return "", err
	}
	return name, nil
}

func cleanTemp(f *os.File, name string) {
	if err := f.Close(); err != nil {
		if rmErr := os.Remove(name); rmErr != nil {
			return
		}
		return
	}
	if rmErr := os.Remove(name); rmErr != nil {
		return
	}
}

func syncDirectory(dir string) error {
	dirFD, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := dirFD.Close(); closeErr != nil {
			return
		}
	}()
	return dirFD.Sync()
}
