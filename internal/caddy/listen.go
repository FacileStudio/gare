package caddy

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

// listenerTimeout bounds a single ingress probe so a firewalled or hung port cannot stall a deploy.
const listenerTimeout = 2 * time.Second

// ListenerBound reports whether a TCP listener accepts connections on the local port.
func ListenerBound(ctx context.Context, port int) bool {
	dialer := net.Dialer{Timeout: listenerTimeout}
	conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
	if err != nil {
		return false
	}
	defer conn.Close()
	return true
}

// VerifyListening reports whether anything accepts connections on at least one of the ports. A
// snippet on disk is not proof the running server serves it, so a deploy of an app with domains
// calls this after the reload and fails when nothing answers.
func VerifyListening(ctx context.Context, ports []int) error {
	if len(ports) == 0 {
		return nil
	}
	missing := make([]string, 0, len(ports))
	for _, port := range ports {
		if ListenerBound(ctx, port) {
			return nil
		}
		missing = append(missing, strconv.Itoa(port))
	}
	return fmt.Errorf("no ingress is listening on port(s) %s", strings.Join(missing, ", "))
}
