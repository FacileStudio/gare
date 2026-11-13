// Package xdg resolves the XDG base directories for the current user.
package xdg

import (
	"os"
	"os/user"
	"path/filepath"
)

// Home returns the current user's home directory, falling back to the passwd entry and HOME.
func Home() string {
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		return home
	}
	if entry, err := user.Current(); err == nil && entry.HomeDir != "" {
		return entry.HomeDir
	}
	return os.Getenv("HOME")
}

// ConfigHome returns the XDG configuration directory, defaulting to ~/.config.
func ConfigHome() string {
	if configHome := os.Getenv("XDG_CONFIG_HOME"); configHome != "" {
		return configHome
	}
	return filepath.Join(Home(), ".config")
}

// DataHome returns the XDG data directory, defaulting to ~/.local/share.
func DataHome() string {
	if dataHome := os.Getenv("XDG_DATA_HOME"); dataHome != "" {
		return dataHome
	}
	return filepath.Join(Home(), ".local", "share")
}
