package health

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// Probe describes a single HTTP readiness probe for a workload. AnyAnswer marks the implicit
// probe gare adds when nothing is declared: it only proves the port speaks HTTP, since an API with
// no root route legitimately answers 404 at `/`. It is never serialised, so a stored or declared
// probe always keeps the 2xx/3xx rule.
type Probe struct {
	Name      string `yaml:"name,omitempty" json:"name,omitempty"`
	Port      int    `yaml:"port" json:"port"`
	Path      string `yaml:"path,omitempty" json:"path,omitempty"`
	AnyAnswer bool   `yaml:"-" json:"-"`
}

// Label returns a human readable identifier for the probe.
func (p Probe) Label() string {
	if p.Name != "" {
		return p.Name
	}
	return fmt.Sprintf("port %d", p.Port)
}

// Accepts reports whether status counts as healthy for the probe: any status for an implicit
// probe, a 2xx or 3xx for a declared one.
func (p Probe) Accepts(status int) bool {
	return p.AnyAnswer || (status >= 200 && status < 400)
}

// Verify repeatedly sends HTTP GET requests to targetURL until the probe accepts the answer.
func Verify(ctx context.Context, probe Probe, targetURL string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	client := &http.Client{Timeout: 2 * time.Second}
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	var lastErr error
	for {
		lastErr = executeProbe(ctx, client, probe, targetURL)
		if lastErr == nil {
			return nil
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("health check timed out for %s: %w", targetURL, lastErr)
		case <-ticker.C:
		}
	}
}

// executeProbe sends one GET request and turns an answer the probe rejects into an error.
func executeProbe(ctx context.Context, client *http.Client, probe Probe, targetURL string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if probe.Accepts(resp.StatusCode) {
		return nil
	}
	return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
}
