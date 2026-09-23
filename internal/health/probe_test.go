package health

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestProbeSuccess(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer ts.Close()

	ctx := context.Background()
	if err := Verify(ctx, ts.URL, 2*time.Second); err != nil {
		t.Fatalf("expected probe success, got error: %v", err)
	}
}

func TestProbeRetryUntilSuccess(t *testing.T) {
	var count atomic.Int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if count.Add(1) < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	ctx := context.Background()
	if err := Verify(ctx, ts.URL, 3*time.Second); err != nil {
		t.Fatalf("expected probe to succeed after retries, got: %v", err)
	}
}

func TestProbeTimeout(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	ctx := context.Background()
	if err := Verify(ctx, ts.URL, 300*time.Millisecond); err == nil {
		t.Fatalf("expected probe timeout error, got nil")
	}
}

func TestProbeLabel(t *testing.T) {
	if got := (Probe{Name: "api", Port: 8100}).Label(); got != "api" {
		t.Errorf("Label with name: got %q", got)
	}
	if got := (Probe{Port: 8100}).Label(); got != "port 8100" {
		t.Errorf("Label without name: got %q", got)
	}
}
