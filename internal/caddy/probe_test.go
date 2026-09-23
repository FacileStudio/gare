package caddy

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
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

func serveOn(t *testing.T, server *httptest.Server) IngressTarget {
	t.Helper()
	addr, ok := server.Listener.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("unexpected server address %v", server.Listener.Addr())
	}
	return IngressTarget{Host: "app.example.com", Port: addr.Port, Secure: server.TLS != nil}
}

// TestProbeIngressAnswersWithTheHostname pins that the ingress is addressed by the hostname the app
// is configured for, not by the loopback socket the probe dials.
func TestProbeIngressAnswersWithTheHostname(t *testing.T) {
	var host string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host = r.Host
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	if err := ProbeIngress(context.Background(), serveOn(t, server)); err != nil {
		t.Fatalf("expected the ingress to answer, got: %v", err)
	}
	if host != "app.example.com" {
		t.Fatalf("expected the hostname as the Host header, got: %q", host)
	}
}

// TestProbeIngressAnswersOverTLS pins that a hostname served with a certificate this host does not
// trust is answered rather than refused: the probe asks whether the ingress answers, not whether its
// certificate would satisfy a client.
func TestProbeIngressAnswersOverTLS(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	target := serveOn(t, server)
	if !target.Secure {
		t.Fatal("expected the TLS server to be probed over TLS")
	}
	if err := ProbeIngress(context.Background(), target); err != nil {
		t.Fatalf("expected a self-signed ingress to answer, got: %v", err)
	}
}

func TestProbeIngressReportsNoAnswer(t *testing.T) {
	port, listener := openPort(t)
	if err := listener.Close(); err != nil {
		t.Fatalf("failed to close listener: %v", err)
	}

	err := ProbeIngress(context.Background(), IngressTarget{Host: "app.example.com", Port: port})
	if err == nil {
		t.Fatalf("expected a closed port %d to fail the probe", port)
	}
	for _, needle := range []string{"app.example.com", strconv.Itoa(port)} {
		if !strings.Contains(err.Error(), needle) {
			t.Fatalf("expected the error to mention %q, got: %v", needle, err)
		}
	}
}

// TestProbeIngressRejectsASocketThatDoesNotAnswer separates listening from answering: the listener
// accepts the connection and closes it, which leaves the hostname unserved.
func TestProbeIngressRejectsASocketThatDoesNotAnswer(t *testing.T) {
	port, listener := openPort(t)
	defer listener.Close()
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()

	if err := ProbeIngress(context.Background(), IngressTarget{Host: "app.example.com", Port: port}); err == nil {
		t.Fatal("expected a socket that does not answer HTTP to fail the probe")
	}
}

// TestProbeIngressKeepsTheFirstResponse pins that a redirect is the answer: following it would send
// the probe to the public hostname instead of the local ingress.
func TestProbeIngressKeepsTheFirstResponse(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		http.Redirect(w, r, "https://app.example.com/elsewhere", http.StatusMovedPermanently)
	}))
	defer server.Close()

	if err := ProbeIngress(context.Background(), serveOn(t, server)); err != nil {
		t.Fatalf("expected the redirect itself to answer, got: %v", err)
	}
	if requests != 1 {
		t.Fatalf("expected the probe to stop at the first response, got %d requests", requests)
	}
}
