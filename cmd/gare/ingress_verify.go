package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/FacileStudio/gare/internal/caddy"
	"github.com/FacileStudio/gare/internal/storage"
)

// ingressPorts returns the local ports an app's domains are served on. A hostname without an
// explicit port is served on the standard ingress ports, because that is where a gare-written
// snippet binds when its site address names no port.
func ingressPorts(domains []string) []int {
	ports := make([]int, 0, len(domains))
	seen := make(map[int]bool, len(domains))
	for _, domain := range domains {
		port := storage.DomainPort(domain)
		if port > 0 && !seen[port] {
			seen[port] = true
			ports = append(ports, port)
		}
	}
	if len(ports) == 0 {
		return []int{80, 443}
	}
	return ports
}

// verifyIngress proves the running ingress serves an app's domains, because a snippet on disk is no
// evidence the server accepted it. An app reachable on its own port alone is skipped.
func verifyIngress(ctx context.Context, cfg *storage.AppConfig) error {
	if len(cfg.Domains) == 0 {
		return nil
	}
	ports := ingressPorts(cfg.Domains)
	if err := caddy.VerifyListening(ctx, ports); err != nil {
		return fmt.Errorf("%s is configured for %s but %v, so it is not reachable through the ingress",
			cfg.Name, strings.Join(cfg.Domains, ", "), err)
	}
	printVerbose(ctx, "Ingress verified: listening on %s for %s", formatPorts(ports), strings.Join(cfg.Domains, ", "))
	return nil
}

func formatPorts(ports []int) string {
	values := make([]string, 0, len(ports))
	for _, port := range ports {
		values = append(values, strconv.Itoa(port))
	}
	return strings.Join(values, ", ")
}
