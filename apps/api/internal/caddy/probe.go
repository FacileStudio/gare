package caddy

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"
)

// ingressProbeTimeout bounds a single ingress probe so a firewalled or hung port cannot stall a deploy.
const ingressProbeTimeout = 2 * time.Second

// IngressTarget is one hostname and loopback listener the ingress is asked to answer on. Secure
// selects TLS, so a probe addresses the socket the hostname is served on instead of the plaintext one.
type IngressTarget struct {
	Host   string
	Port   int
	Secure bool
}

// ProbeIngress sends one HTTP request to the loopback listener the hostname is served on, carrying
// that hostname as the request's Host header, and reports whether the ingress answered it. A bound
// socket is not an answer: a connection nothing replies to fails the probe.
func ProbeIngress(ctx context.Context, target IngressTarget) error {
	client := &http.Client{
		Timeout:       ingressProbeTimeout,
		Transport:     ingressTransport(target),
		CheckRedirect: stopAtFirstResponse,
	}
	defer client.CloseIdleConnections()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ingressURL(target), nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("nothing answered for %s on %s: %w", target.Host, target.address(), err)
	}
	defer resp.Body.Close()
	return nil
}

func stopAtFirstResponse(*http.Request, []*http.Request) error {
	return http.ErrUseLastResponse
}

// ingressTransport dials the loopback listener while the request keeps the hostname as its URL host,
// so the probe reaches the local ingress carrying the Host header and server name a visitor's
// request carries. Certificate verification is off because the probe asks whether the ingress
// answers for the hostname, not whether its certificate is trusted, and a Caddy on this host may
// serve an internal one; the loopback connection is not a trust boundary.
func ingressTransport(target IngressTarget) *http.Transport {
	return &http.Transport{
		DialContext: func(ctx context.Context, network, _ string) (net.Conn, error) {
			dialer := net.Dialer{Timeout: ingressProbeTimeout}
			return dialer.DialContext(ctx, network, target.address())
		},
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
}

// ingressURL addresses the hostname over the scheme the ingress port is served with.
func ingressURL(target IngressTarget) string {
	scheme := "http"
	if target.Secure {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s/", scheme, target.Host)
}

func (t IngressTarget) address() string {
	return net.JoinHostPort("127.0.0.1", strconv.Itoa(t.Port))
}
