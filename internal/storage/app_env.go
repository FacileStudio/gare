package storage

import (
	"bytes"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"sort"

	"github.com/FacileStudio/gare/internal/atomicfile"
	"github.com/FacileStudio/gare/internal/dotenv"
)

// GetAppEnvPath returns the path to the application env file.
func GetAppEnvPath(appDir string) string {
	return filepath.Join(appDir, "env")
}

// GetAppStaticConfigPath returns the Caddyfile path a static workload's container serves.
func GetAppStaticConfigPath(appDir string) string {
	return filepath.Join(appDir, "Caddyfile")
}

// GetAppEnv reads environment variables from the application env file.
func GetAppEnv(appDir string) (map[string]string, error) {
	data, err := os.ReadFile(GetAppEnvPath(appDir))
	if os.IsNotExist(err) {
		return make(map[string]string), nil
	}
	if err != nil {
		return nil, err
	}
	return dotenv.Parse(string(data))
}

// EnsureAppEnvFile creates the application env file when it does not exist yet.
func EnsureAppEnvFile(appDir string) error {
	if err := os.MkdirAll(appDir, 0755); err != nil {
		return err
	}
	path := GetAppEnvPath(appDir)
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return atomicfile.WriteFile(path, nil, 0644)
	}
	if err != nil {
		return err
	}
	return nil
}

// SetAppEnv updates or adds environment variables in the application env file.
func SetAppEnv(appDir string, vars map[string]string) error {
	current, err := GetAppEnv(appDir)
	if err != nil {
		return err
	}
	maps.Copy(current, vars)
	return writeEnvFile(GetAppEnvPath(appDir), current)
}

// UnsetAppEnv removes specified keys from the application env file.
func UnsetAppEnv(appDir string, keys []string) error {
	current, err := GetAppEnv(appDir)
	if err != nil {
		return err
	}
	for _, k := range keys {
		delete(current, k)
	}
	return writeEnvFile(GetAppEnvPath(appDir), current)
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
