package storage

import (
	"maps"
	"os"
	"path/filepath"

	"github.com/FacileStudio/gare/internal/atomicfile"
	"github.com/FacileStudio/gare/internal/dotenv"
)

// envFileMode keeps an application's env file readable by its owner alone, because it holds the
// application's secrets.
const envFileMode = 0600

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
		return atomicfile.WriteFile(path, nil, envFileMode)
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
	return atomicfile.WriteFile(filePath, dotenv.Format(envs), envFileMode)
}
