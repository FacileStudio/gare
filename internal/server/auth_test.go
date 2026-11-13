package server

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestServerWithoutSecretRefusesEverything(t *testing.T) {
	var called atomic.Bool
	s := New(Config{DeployHandler: func(ctx context.Context, appName string) error {
		called.Store(true)
		return nil
	}})
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()

	body := []byte(`{}`)
	for _, header := range [][2]string{
		{"X-Hub-Signature-256", computeSignature("", body)},
		{"X-Gitlab-Token", ""},
		{"", ""},
	} {
		req, _ := http.NewRequest(http.MethodPost, ts.URL+"/webhook/app", bytes.NewReader(body))
		if header[0] != "" {
			req.Header.Set(header[0], header[1])
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("header %q: got %d, want 401", header[0], resp.StatusCode)
		}
	}
	if called.Load() {
		t.Error("a server without a secret ran a deploy")
	}
}
