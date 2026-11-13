package ingress

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
)

// adminAPIURL is Caddy's local configuration endpoint, which reloads a running server in place.
const adminAPIURL = "http://127.0.0.1:2019/load"

// Reload instructs Caddy to reload its configuration using the CLI or Admin API. A reload that
// cannot be delivered is reported rather than swallowed, because the snippet on disk is then not
// the configuration the running server serves.
func Reload(ctx context.Context) error {
	if err := EnsureCaddyfile(); err != nil {
		return err
	}
	caddyfile := ResolveCaddyfilePath()
	cmd := exec.CommandContext(ctx, "caddy", "reload", "--config", caddyfile)
	cliErr := cmd.Run()
	if cliErr == nil {
		return nil
	}
	if apiErr := reloadViaAdminAPI(ctx, adminAPIURL, caddyfile); apiErr != nil {
		return fmt.Errorf("caddy reload failed: %v: %w", cliErr, apiErr)
	}
	return nil
}

func reloadViaAdminAPI(ctx context.Context, url, configPath string) error {
	content, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(content))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "text/caddyfile")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("caddy admin api at %s unreachable: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	body, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("caddy admin api returned %s: %s", resp.Status, string(body))
}
