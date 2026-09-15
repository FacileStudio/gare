package server

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

type zeroReader struct{}

func (zeroReader) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = 0
	}
	return len(p), nil
}

func TestMaxBytesPayload(t *testing.T) {
	var deployCalled atomic.Bool
	s := New(Config{
		DeployHandler: func(ctx context.Context, appName string) error {
			deployCalled.Store(true)
			return nil
		},
	})
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()

	largeBody := io.LimitReader(zeroReader{}, (10<<20)+1024)
	resp, err := http.Post(ts.URL+"/webhook/large-app", "application/json", largeBody)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("got status %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
	if deployCalled.Load() {
		t.Error("deploy handler was called on oversized payload")
	}
}

func TestShutdownWaitsForDeploy(t *testing.T) {
	var deployFinished atomic.Bool
	s := NewServer("0", "", func(ctx context.Context, appName string) error {
		time.Sleep(80 * time.Millisecond)
		deployFinished.Store(true)
		return nil
	})
	go s.Start("127.0.0.1:0")
	addr := waitForServerAddr(s)
	if addr == "" {
		t.Fatal("server address not bound")
	}

	resp, err := http.Post("http://"+addr+"/webhook/app", "application/json", bytes.NewReader([]byte(`{}`)))
	if err != nil {
		t.Fatalf("post error: %v", err)
	}
	resp.Body.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := s.Shutdown(ctx); err != nil {
		t.Fatalf("shutdown error: %v", err)
	}
	if !deployFinished.Load() {
		t.Error("expected deploy to finish before shutdown returned")
	}
}

func TestShutdownTimeout(t *testing.T) {
	release := make(chan struct{})
	s := NewServer("0", "", func(ctx context.Context, appName string) error {
		<-release
		return nil
	})
	go s.Start("127.0.0.1:0")
	addr := waitForServerAddr(s)
	if addr == "" {
		t.Fatal("server address not bound")
	}

	resp, err := http.Post("http://"+addr+"/webhook/app", "application/json", bytes.NewReader([]byte(`{}`)))
	if err != nil {
		t.Fatalf("post error: %v", err)
	}
	resp.Body.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	err = s.Shutdown(ctx)
	close(release)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("got error %v, want DeadlineExceeded", err)
	}
}
