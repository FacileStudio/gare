package caddy

import (
	"context"
	"net"
	"strconv"
	"strings"
	"testing"
)

func openPort(t *testing.T) (int, net.Listener) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to open a listener: %v", err)
	}
	addr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("unexpected listener address %v", listener.Addr())
	}
	return addr.Port, listener
}

func TestListenerBound(t *testing.T) {
	ctx := context.Background()
	port, listener := openPort(t)
	defer listener.Close()
	if !ListenerBound(ctx, port) {
		t.Fatalf("expected port %d to accept connections", port)
	}
	if ListenerBound(ctx, port+1) {
		t.Fatalf("expected port %d to be closed", port+1)
	}
}

func TestVerifyListening(t *testing.T) {
	ctx := context.Background()
	port, listener := openPort(t)
	defer listener.Close()

	if err := VerifyListening(ctx, []int{port}); err != nil {
		t.Fatalf("expected an open port to pass, got: %v", err)
	}
	if err := VerifyListening(ctx, []int{port, 1}); err != nil {
		t.Fatalf("expected one bound port to satisfy the check, got: %v", err)
	}
	if err := VerifyListening(ctx, nil); err != nil {
		t.Fatalf("expected no ports to skip the check, got: %v", err)
	}

	closed, closedListener := openPort(t)
	if err := closedListener.Close(); err != nil {
		t.Fatalf("failed to close listener: %v", err)
	}
	err := VerifyListening(ctx, []int{closed})
	if err == nil {
		t.Fatalf("expected a closed port %d to fail", closed)
	}
	if !strings.Contains(err.Error(), strconv.Itoa(closed)) {
		t.Fatalf("expected the error to name port %d, got: %v", closed, err)
	}
}
