package server

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func computeSignature(secret string, payload []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func TestValidHMACWebhook(t *testing.T) {
	secret := "test-secret-value"
	var deployedApp string
	var deployMu sync.Mutex
	deployDone := make(chan struct{}, 1)

	s := New(Config{
		Secret: secret,
		DeployHandler: func(ctx context.Context, appName string) error {
			deployMu.Lock()
			deployedApp = appName
			deployMu.Unlock()
			deployDone <- struct{}{}
			return nil
		},
	})
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()

	payload := []byte(`{"event":"deploy"}`)
	postHMACWebhook(t, ts.URL+"/webhook/my-service", payload, computeSignature(secret, payload))

	select {
	case <-deployDone:
		deployMu.Lock()
		defer deployMu.Unlock()
		if deployedApp != "my-service" {
			t.Errorf("got deployed app %q, want %q", deployedApp, "my-service")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for deploy handler")
	}
}

func postHMACWebhook(t *testing.T, targetURL string, payload []byte, sig string) {
	req, err := http.NewRequest(http.MethodPost, targetURL, bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("X-Hub-Signature-256", sig)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d reading body failed: %v", resp.StatusCode, err)
	}
	if string(body) != "Deploy queued for my-service" {
		t.Errorf("got body %q, want %q", string(body), "Deploy queued for my-service")
	}
}

func TestInvalidHMACSignature(t *testing.T) {
	secret := "test-secret-value"
	handlerCalled := false
	s := New(Config{
		Secret: secret,
		DeployHandler: func(ctx context.Context, appName string) error {
			handlerCalled = true
			return nil
		},
	})
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()

	payload := []byte(`{"event":"deploy"}`)
	tests := []struct{ name, header, value string }{
		{"missing signature header", "", ""},
		{"wrong signature", "X-Hub-Signature-256", "sha256=0000000000000000000000000000000000000000000000000000000000000000"},
		{"missing sha256 prefix", "X-Hub-Signature-256", computeSignature(secret, payload)[len("sha256="):]},
		{"wrong secret signature", "X-Hub-Signature-256", computeSignature("wrong-secret", payload)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assertWebhookUnauthorized(t, ts.URL+"/webhook/app", payload, tc.header, tc.value)
		})
	}
	if handlerCalled {
		t.Error("handler was unexpectedly called")
	}
}

func assertWebhookUnauthorized(t *testing.T, targetURL string, payload []byte, header, value string) {
	req, err := http.NewRequest(http.MethodPost, targetURL, bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	if header != "" {
		req.Header.Set(header, value)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("got status %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}
}

func TestGitLabTokenHeader(t *testing.T) {
	secret := "gitlab-secret-token"
	deployDone := make(chan struct{}, 1)
	s := New(Config{
		Secret: secret,
		DeployHandler: func(ctx context.Context, appName string) error {
			deployDone <- struct{}{}
			return nil
		},
	})
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()

	payload := []byte(`{"event":"push"}`)
	assertWebhookUnauthorized(t, ts.URL+"/webhook/gitlab-app", payload, "X-Gitlab-Token", "invalid-token")

	req, err := http.NewRequest(http.MethodPost, ts.URL+"/webhook/gitlab-app", bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("X-Gitlab-Token", secret)
	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("request failed or status not ok: %v", err)
	}
	defer resp.Body.Close()

	select {
	case <-deployDone:
	case <-time.After(2 * time.Second):
		t.Fatal("deploy handler not triggered for valid gitlab token")
	}
}

func postPathCase(t *testing.T, targetURL string, body []byte, expectedStatus int) {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	resp, err := http.Post(targetURL, "application/json", reader)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != expectedStatus {
		t.Errorf("got status %d, want %d", resp.StatusCode, expectedStatus)
	}
}

func TestPathParsingAndFallback(t *testing.T) {
	var deployedApps []string
	var mu sync.Mutex
	var wg sync.WaitGroup

	s := New(Config{
		DeployHandler: func(ctx context.Context, appName string) error {
			mu.Lock()
			deployedApps = append(deployedApps, appName)
			mu.Unlock()
			wg.Done()
			return nil
		},
	})
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()

	wg.Add(3)
	postPathCase(t, ts.URL+"/webhook/service-alpha", nil, http.StatusOK)
	postPathCase(t, ts.URL+"/webhook/service-beta", []byte(`{"app":"should-ignore"}`), http.StatusOK)
	postPathCase(t, ts.URL+"/webhook", []byte(`{"app":"service-gamma"}`), http.StatusOK)
	postPathCase(t, ts.URL+"/webhook", []byte(`{"event":"deploy"}`), http.StatusBadRequest)
	wg.Wait()

	mu.Lock()
	apps := append([]string(nil), deployedApps...)
	mu.Unlock()
	expected := []string{"service-alpha", "service-beta", "service-gamma"}
	if len(apps) != len(expected) || apps[0] != expected[0] || apps[1] != expected[1] || apps[2] != expected[2] {
		t.Fatalf("got apps %v, want %v", apps, expected)
	}
}

func TestPathTraversalAppNames(t *testing.T) {
	s := New(Config{})
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()

	postPathCase(t, ts.URL+"/webhook", []byte(`{"app":"../../bad"}`), http.StatusBadRequest)
	postPathCase(t, ts.URL+"/webhook", []byte(`{"app":"../etc"}`), http.StatusBadRequest)
	postPathCase(t, ts.URL+"/webhook/invalid@name", nil, http.StatusBadRequest)

	req := httptest.NewRequest(http.MethodPost, "/webhook", nil)
	req.SetPathValue("app", "../../etc")
	rec := httptest.NewRecorder()
	s.handleWebhook(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("got status %d, want %d", rec.Code, http.StatusBadRequest)
	}
}
