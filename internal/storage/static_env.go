package storage

import (
	"bytes"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"sort"

	"github.com/FacileStudio/gare/internal/atomicfile"
)

// GetStaticEnvPath returns the path to the static env file.
func GetStaticEnvPath(appDir string) string {
	return filepath.Join(appDir, "env")
}

// GetStaticEnv reads environment variables from a static app env file.
func GetStaticEnv(appDir string) (map[string]string, error) {
	path := GetStaticEnvPath(appDir)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return make(map[string]string), nil
	}
	if err != nil {
		return nil, err
	}
	return ParseDotEnv(string(data))
}

// SetStaticEnv updates or adds environment variables in the static app env file.
func SetStaticEnv(appDir string, vars map[string]string) error {
	current, err := GetStaticEnv(appDir)
	if err != nil {
		return err
	}
	maps.Copy(current, vars)
	return writeEnvFile(GetStaticEnvPath(appDir), current)
}

// UnsetStaticEnv removes specified keys from the static app env file.
func UnsetStaticEnv(appDir string, keys []string) error {
	current, err := GetStaticEnv(appDir)
	if err != nil {
		return err
	}
	for _, k := range keys {
		delete(current, k)
	}
	return writeEnvFile(GetStaticEnvPath(appDir), current)
}

func writeEnvFile(filePath string, envs map[string]string) error {
	keys := make([]string, 0, len(envs))
	for k := range envs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var buf bytes.Buffer
	for _, k := range keys {
		fmt.Fprintf(&buf, "%s=%s\n", k, envs[k])
	}
	return atomicfile.WriteFile(filePath, buf.Bytes(), 0644)
}
