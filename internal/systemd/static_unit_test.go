package systemd

import (
	"os"
	"strings"
	"testing"
)

func TestGenerateStaticUnit(t *testing.T) {
	content, err := GenerateStaticUnit(withPodmanPath(StaticSiteUnit(
		"my-site", 8100, "/srv/repo/dist", "/home/user/.local/share/gare/apps/my-site/env",
		"/home/user/.local/share/gare/apps/my-site/Caddyfile")))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedSnippets := []string{
		"Description=Gare Managed Static App: my-site",
		"RequiresMountsFor=/srv/repo/dist",
		"RequiresMountsFor=/home/user/.local/share/gare/apps/my-site/Caddyfile",
		"-v /srv/repo/dist:/srv:ro,Z",
		"-v /home/user/.local/share/gare/apps/my-site/Caddyfile:/etc/caddy/Caddyfile:ro,Z",
		"--publish 8100:80",
		"--env-file /home/user/.local/share/gare/apps/my-site/env",
		"docker.io/library/caddy:2-alpine /usr/bin/caddy run --config /etc/caddy/Caddyfile --adapter caddyfile",
		"Type=notify",
		"NotifyAccess=all",
		"Restart=on-failure",
		"[Install]",
		"WantedBy=default.target",
	}
	for _, snippet := range expectedSnippets {
		if !strings.Contains(content, snippet) {
			t.Errorf("expected unit to contain %q, got:\n%s", snippet, content)
		}
	}
	if strings.Contains(content, `Volume="`) || strings.Contains(content, `EnvironmentFile="`) {
		t.Errorf("systemd does not re-quote these values, quoting breaks the generated mount:\n%s", content)
	}
}

func TestGenerateStaticUnitRunsTheServerBinaryNotTheImageCmd(t *testing.T) {
	content, err := GenerateStaticUnit(withPodmanPath(StaticSiteUnit("my-site", 8100, "/srv/dist", "/srv/env", "/srv/Caddyfile")))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(content, "--entrypoint") {
		t.Errorf("the image needs no entrypoint override, got:\n%s", content)
	}
	if !strings.Contains(content, "caddy:2-alpine /usr/bin/caddy run --config /etc/caddy/Caddyfile --adapter caddyfile") {
		t.Errorf("the caddy image carries its invocation in Cmd, so arguments appended after the image replace it and the runtime execs a binary named run, got:\n%s", content)
	}
}

func TestGenerateStaticUnitTearsDownThroughTheCidfile(t *testing.T) {
	content, err := GenerateStaticUnit(withPodmanPath(StaticSiteUnit("my-site", 8100, "/srv/dist", "/srv/env", "/srv/Caddyfile")))
	if err != nil {
		t.Fatal(err)
	}
	for _, snippet := range []string{
		"--cidfile=%t/%N.cid --replace --rm",
		"ExecStop=/usr/bin/podman rm -v -f -i --cidfile=%t/%N.cid",
		"ExecStopPost=-/usr/bin/podman rm -v -f -i --cidfile=%t/%N.cid",
	} {
		if !strings.Contains(content, snippet) {
			t.Errorf("expected unit to contain %q, got:\n%s", snippet, content)
		}
	}
}

func TestGenerateStaticUnitRejectsBadInput(t *testing.T) {
	configFile := "/srv/Caddyfile"
	cases := map[string]StaticUnitData{
		"relative root":      StaticSiteUnit("my-site", 8100, "dist", "/srv/env", configFile),
		"root with space":    StaticSiteUnit("my-site", 8100, "/srv/my dist", "/srv/env", configFile),
		"quoted root":        StaticSiteUnit("my-site", 8100, `/srv/"dist"`, "/srv/env", configFile),
		"relative env file":  StaticSiteUnit("my-site", 8100, "/srv/dist", "env", configFile),
		"relative caddyfile": StaticSiteUnit("my-site", 8100, "/srv/dist", "/srv/env", "Caddyfile"),
		"missing port":       StaticSiteUnit("my-site", 0, "/srv/dist", "/srv/env", configFile),
		"out of range port":  StaticSiteUnit("my-site", 70000, "/srv/dist", "/srv/env", configFile),
	}
	for name, data := range cases {
		if _, err := GenerateStaticUnit(data); err == nil {
			t.Errorf("expected an error for %s", name)
		}
	}
}

func TestWriteStaticUnitRejectsInvalidInput(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := WriteStaticUnit(StaticSiteUnit("my-site", 8100, "dist", "/srv/env", "/srv/Caddyfile")); err == nil {
		t.Error("expected an error for a relative static root")
	}
	if _, err := os.Stat(GetUnitPath("my-site")); !os.IsNotExist(err) {
		t.Error("an invalid unit must not be written")
	}
}

func TestGeneratedStaticUnitIsAcceptedBySystemd(t *testing.T) {
	unitDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", unitDir)
	rootDir := t.TempDir()
	configFile := rootDir + "/Caddyfile"
	envFile := rootDir + "/env"
	for _, path := range []string{configFile, envFile} {
		if err := os.WriteFile(path, []byte(""), 0644); err != nil {
			t.Fatal(err)
		}
	}

	if err := WriteStaticUnit(StaticSiteUnit("my-site", 8100, rootDir, envFile, configFile)); err != nil {
		t.Fatal(err)
	}
	verifyUnitWithSystemd(t, GetUnitPath("my-site"))
}

func withPodmanPath(data StaticUnitData) StaticUnitData {
	data.PodmanPath = "/usr/bin/podman"
	return data
}
