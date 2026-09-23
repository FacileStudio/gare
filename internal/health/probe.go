package health

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// Probe describes a single HTTP readiness probe for a workload.
type Probe struct {
	Name string `yaml:"name,omitempty" json:"name,omitempty"`
	Port int    `yaml:"port" json:"port"`
	Path string `yaml:"path,omitempty" json:"path,omitempty"`
}

// Label returns a human readable identifier for the probe.
func (p Probe) Label() string {
	if p.Name != "" {
		return p.Name
	}
	return fmt.Sprintf("port %d", p.Port)
}

// Verify repeatedly sends HTTP GET requests to targetURL until a 2xx or 3xx status is returned.
func Verify(ctx context.Context, targetURL string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	client := &http.Client{Timeout: 2 * time.Second}
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	var lastErr error
	for {
		lastErr = executeProbe(ctx, client, targetURL)
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

func executeProbe(ctx context.Context, client *http.Client, targetURL string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		return nil
	}
	return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
}
