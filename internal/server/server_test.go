package server

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestHealthCheck(t *testing.T) {
	s := New(Config{})
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/health")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil || resp.StatusCode != http.StatusOK || string(body) != "OK" {
		t.Errorf("status %d body %q", resp.StatusCode, string(body))
	}

	postResp, err := http.Post(ts.URL+"/health", "text/plain", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer postResp.Body.Close()
	if postResp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("got status %d, want %d", postResp.StatusCode, http.StatusMethodNotAllowed)
	}
}

func TestServerLifecycle(t *testing.T) {
	s := NewServer("0", "", func(ctx context.Context, appName string) error {
		return nil
	})
	errCh := make(chan error, 1)
	go func() {
		errCh <- s.Start("127.0.0.1:0")
	}()

	addr := waitForServerAddr(s)
	if addr == "" {
		t.Fatal("server address was not bound")
	}

	assertHealthEndpoint(t, addr)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := s.Shutdown(ctx); err != nil {
		t.Fatalf("shutdown failed: %v", err)
	}

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("start returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for Start to return")
	}
}

func waitForServerAddr(s *Server) string {
	for range 50 {
		time.Sleep(20 * time.Millisecond)
		if addr := s.Addr(); addr != "" {
			return addr
		}
	}
	return ""
}

func assertHealthEndpoint(t *testing.T, addr string) {
	resp, err := http.Get(fmt.Sprintf("http://%s/health", addr))
	if err != nil {
		t.Fatalf("failed to reach server: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("got status %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

func TestPerAppConcurrencyLock(t *testing.T) {
	var currentApp1, maxApp1, app1Runs atomic.Int32
	var wg sync.WaitGroup

	s := New(Config{
		DeployHandler: func(ctx context.Context, appName string) error {
			defer wg.Done()
			if appName == "app1" {
				recordConcurrency(&currentApp1, &maxApp1, &app1Runs)
			}
			return nil
		},
	})

	ts := httptest.NewServer(s.Handler())
	defer ts.Close()

	total := 6
	wg.Add(total)
	for range total {
		go sendTestWebhook(t, ts.URL+"/webhook/app1")
	}
	wg.Wait()

	if max := maxApp1.Load(); max != 1 {
		t.Errorf("expected max concurrency 1 for app1, got %d", max)
	}
	if runs := app1Runs.Load(); runs != int32(total) {
		t.Errorf("expected %d runs, got %d", total, runs)
	}
}

func recordConcurrency(curr, maxVal, runs *atomic.Int32) {
	c := curr.Add(1)
	for {
		m := maxVal.Load()
		if c <= m || maxVal.CompareAndSwap(m, c) {
			break
		}
	}
	time.Sleep(30 * time.Millisecond)
	curr.Add(-1)
	runs.Add(1)
}

func sendTestWebhook(t *testing.T, targetURL string) {
	resp, err := http.Post(targetURL, "application/json", bytes.NewReader([]byte(`{}`)))
	if err != nil {
		t.Errorf("post error: %v", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("got status %d, want %d", resp.StatusCode, http.StatusOK)
	}
}
