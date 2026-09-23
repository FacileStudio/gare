package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/FacileStudio/gare/internal/caddy"
	"github.com/FacileStudio/gare/internal/storage"
)

// ingressTarget names one loopback listener the ingress should answer on for an app's domain.
type ingressTarget struct {
	host   string
	port   int
	secure bool
}

// ingressTargets pairs a domain with the listeners it is served on: the port it names, or the
// standard ingress pair when it names none, because that is where a gare-written snippet binds when
// its site address carries no port.
func ingressTargets(domains []string) []ingressTarget {
	targets := make([]ingressTarget, 0, len(domains)*2)
	for _, domain := range domains {
		host, port := storage.DomainHostPort(domain)
		if port > 0 {
			targets = append(targets, ingressTarget{host: host, port: port, secure: port == 443})
			continue
		}
		targets = append(targets, ingressTarget{host: host, port: 80}, ingressTarget{host: host, port: 443, secure: true})
	}
	return targets
}

// probeDomain asks the ingress to answer for one domain on each listener it is served on and reports
// the last refusal. A domain naming no port is served on both standard ports, so one answering is
// enough.
func probeDomain(ctx context.Context, domain string) error {
	var lastErr error
	for _, target := range ingressTargets([]string{domain}) {
		probe := caddy.IngressTarget{Host: target.host, Port: target.port, Secure: target.secure}
		lastErr = caddy.ProbeIngress(ctx, probe)
		if lastErr == nil {
			return nil
		}
	}
	return lastErr
}

// verifyIngress proves the running ingress answers for every hostname an app is configured with,
// because a snippet on disk is no evidence the server accepted it and a bound port says nothing
// about which hostnames it serves. A probe is not a health check: it reports whether the ingress
// answered, never what it answered, since an API served at the root of its own hostname may answer
// 404 and still be deployed correctly. An app with no domains is skipped.
func verifyIngress(ctx context.Context, cfg *storage.AppConfig) error {
	if len(cfg.Domains) == 0 {
		return nil
	}
	unanswered := make([]string, 0, len(cfg.Domains))
	var cause error
	for _, domain := range cfg.Domains {
		err := probeDomain(ctx, domain)
		if err == nil {
			continue
		}
		unanswered = append(unanswered, domain)
		if cause == nil {
			cause = err
		}
	}
	if len(unanswered) > 0 {
		return fmt.Errorf("%s is not reachable through the ingress: no answer for %s: %w",
			cfg.Name, strings.Join(unanswered, ", "), cause)
	}
	printVerbose(ctx, "Ingress answered for every domain of %s: %s", cfg.Name, strings.Join(cfg.Domains, ", "))
	return nil
}
