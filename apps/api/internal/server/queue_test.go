package server

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func waitQueueIdle(t *testing.T, s *Server) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		s.queue.mu.Lock()
		idle := len(s.queue.apps) == 0
		s.queue.mu.Unlock()
		if idle {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("queue never went idle")
}

func waitQueueClosed(s *Server) {
	for {
		s.queue.mu.Lock()
		closed := s.queue.closed
		s.queue.mu.Unlock()
		if closed {
			return
		}
		time.Sleep(time.Millisecond)
	}
}

func postAnswer(t *testing.T, targetURL string) (int, string) {
	t.Helper()
	resp, err := signedPost(targetURL, []byte(`{}`))
	if err != nil {
		t.Fatalf("post error: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(body)
}

func TestFiveDeliveriesCoalesce(t *testing.T) {
	release := make(chan struct{})
	var calls atomic.Int32
	s := newSignedServer(func(ctx context.Context, appName string) error {
		calls.Add(1)
		<-release
		return nil
	})
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()

	want := []string{answerQueued, answerRunningNext, answerAlreadyNext, answerAlreadyNext, answerAlreadyNext}
	for i, w := range want {
		status, body := postAnswer(t, ts.URL+"/webhook/app")
		if status != http.StatusAccepted || body != w {
			t.Errorf("delivery %d: got %d %q, want 202 %q", i, status, body, w)
		}
	}
	close(release)
	waitQueueIdle(t, s)
	if got := calls.Load(); got != 2 {
		t.Fatalf("got %d deploys, want 2", got)
	}
}

func TestParallelDeploysAreCapped(t *testing.T) {
	var current, peak, runs atomic.Int32
	s := New(Config{Secret: testSecret, MaxParallel: 2, DeployHandler: func(ctx context.Context, appName string) error {
		recordConcurrency(&current, &peak, &runs)
		return nil
	}})
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()

	for _, app := range []string{"one", "two", "three"} {
		if status, body := postAnswer(t, ts.URL+"/webhook/"+app); status != http.StatusAccepted {
			t.Fatalf("%s: got %d %q", app, status, body)
		}
	}
	waitQueueIdle(t, s)
	if got := peak.Load(); got > 2 {
		t.Errorf("got %d concurrent deploys, want at most 2", got)
	}
	if got := runs.Load(); got != 3 {
		t.Errorf("got %d deploys, want 3", got)
	}
}

func TestShutdownDropsPendingDeploy(t *testing.T) {
	release := make(chan struct{})
	var calls, finished atomic.Int32
	s := newSignedServer(func(ctx context.Context, appName string) error {
		calls.Add(1)
		<-release
		finished.Add(1)
		return nil
	})
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()

	postAnswer(t, ts.URL+"/webhook/app")
	if _, body := postAnswer(t, ts.URL+"/webhook/app"); body != answerRunningNext {
		t.Fatalf("got %q, want %q", body, answerRunningNext)
	}
	done := make(chan error, 1)
	go func() { done <- s.Shutdown(context.Background()) }()
	waitQueueClosed(s)
	close(release)
	if err := <-done; err != nil {
		t.Fatalf("shutdown: %v", err)
	}
	if calls.Load() != 1 || finished.Load() != 1 {
		t.Fatalf("got %d started and %d finished, want the running deploy alone", calls.Load(), finished.Load())
	}
}
