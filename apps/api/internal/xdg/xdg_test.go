package xdg

import (
	"os/user"
	"path/filepath"
	"testing"
)

func TestHomeReturnsTheHomeEnvVar(t *testing.T) {
	t.Setenv("HOME", "/home/example")

	if got := Home(); got != "/home/example" {
		t.Errorf("Home() = %q, want %q", got, "/home/example")
	}
}

func TestHomeFallsBackToThePasswdEntry(t *testing.T) {
	t.Setenv("HOME", "")
	entry, err := user.Current()
	if err != nil || entry.HomeDir == "" {
		t.Skipf("no passwd entry to fall back to: %v", err)
	}

	if got := Home(); got != entry.HomeDir {
		t.Errorf("Home() = %q, want the passwd entry %q", got, entry.HomeDir)
	}
}

func TestConfigHomePrefersItsEnvVar(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/xdg/config")

	if got := ConfigHome(); got != "/xdg/config" {
		t.Errorf("ConfigHome() = %q, want %q", got, "/xdg/config")
	}
}

func TestConfigHomeFallsBackToTheHomeDirectory(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", "/home/example")

	if got, want := ConfigHome(), filepath.Join("/home/example", ".config"); got != want {
		t.Errorf("ConfigHome() = %q, want %q", got, want)
	}
}

func TestDataHomePrefersItsEnvVar(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", "/xdg/data")

	if got := DataHome(); got != "/xdg/data" {
		t.Errorf("DataHome() = %q, want %q", got, "/xdg/data")
	}
}

func TestDataHomeFallsBackToTheHomeDirectory(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("HOME", "/home/example")

	if got, want := DataHome(), filepath.Join("/home/example", ".local", "share"); got != want {
		t.Errorf("DataHome() = %q, want %q", got, want)
	}
}

func TestResolvedPathsStayAbsolute(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("HOME", "/home/example")

	for name, got := range map[string]string{"ConfigHome": ConfigHome(), "DataHome": DataHome()} {
		if !filepath.IsAbs(got) {
			t.Errorf("%s() = %q, want an absolute path", name, got)
		}
	}
}
