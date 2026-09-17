package caddy

import (
	"os"
	"testing"
)

func TestResolveConfDir(t *testing.T) {
	orig := os.Getenv("GARE_CADDY_CONF_DIR")
	defer os.Setenv("GARE_CADDY_CONF_DIR", orig)

	os.Unsetenv("GARE_CADDY_CONF_DIR")
	if got := ResolveConfDir(); got != DefaultConfDir {
		t.Errorf("got %q, want %q", got, DefaultConfDir)
	}

	os.Setenv("GARE_CADDY_CONF_DIR", "/custom/conf.d")
	if got := ResolveConfDir(); got != "/custom/conf.d" {
		t.Errorf("got %q, want /custom/conf.d", got)
	}
}

func TestResolveCaddyfilePath(t *testing.T) {
	orig := os.Getenv("GARE_CADDYFILE")
	defer os.Setenv("GARE_CADDYFILE", orig)

	os.Unsetenv("GARE_CADDYFILE")
	if got := ResolveCaddyfilePath(); got != DefaultCaddyfile {
		t.Errorf("got %q, want %q", got, DefaultCaddyfile)
	}

	os.Setenv("GARE_CADDYFILE", "/custom/Caddyfile")
	if got := ResolveCaddyfilePath(); got != "/custom/Caddyfile" {
		t.Errorf("got %q, want /custom/Caddyfile", got)
	}
}
