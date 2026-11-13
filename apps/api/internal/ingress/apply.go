package ingress

import (
	"context"
	"fmt"
	"strings"

	"github.com/FacileStudio/gare/apps/api/internal/storage"
)

// Apply writes the app's snippet when it has domains and a port and removes it otherwise, reloads
// Caddy and rolls the snippet back when Caddy refuses it. It then proves the running ingress answers
// for every hostname the app is configured with, because a snippet on disk is no evidence the server
// accepted it and a bound port says nothing about which hostnames it serves.
func (i *Ingress) Apply(ctx context.Context, app *storage.AppConfig) error {
	change := func() error { return RemoveSnippet(i.ConfDir, app.Name) }
	if len(app.Domains) > 0 && app.Port > 0 {
		change = func() error { return WriteSnippet(i.ConfDir, app.Name, app.Domains, app.Port) }
	}
	if err := i.commit(ctx, app.Name, change); err != nil {
		return err
	}
	return i.verify(ctx, app)
}

// Remove deletes the app's snippet and reloads Caddy, putting the snippet back when Caddy refuses
// the configuration without it, so the files on disk stay a configuration Caddy accepts.
func (i *Ingress) Remove(ctx context.Context, name string) error {
	return i.commit(ctx, name, func() error { return RemoveSnippet(i.ConfDir, name) })
}

// Targets pairs each domain with the listeners it is served on: the port it names, or the standard
// ingress pair when it names none, because that is where a gare-written snippet binds when its site
// address carries no port.
func Targets(domains []string) []Target {
	targets := make([]Target, 0, len(domains)*2)
	for _, domain := range domains {
		host, port := storage.DomainHostPort(domain)
		if port > 0 {
			targets = append(targets, Target{Host: host, Port: port, Secure: port == 443})
			continue
		}
		targets = append(targets, Target{Host: host, Port: 80}, Target{Host: host, Port: 443, Secure: true})
	}
	return targets
}

// probeDomain asks the ingress to answer for one domain on each listener it is served on and
// reports the last refusal. A domain naming no port is served on both standard ports, so one
// answering is enough.
func (i *Ingress) probeDomain(ctx context.Context, domain string) error {
	var lastErr error
	for _, target := range Targets([]string{domain}) {
		lastErr = i.Probe(ctx, target)
		if lastErr == nil {
			return nil
		}
	}
	return lastErr
}

// verify reports whether the ingress answered for every hostname of the app, never what it
// answered: an API served at the root of its own hostname may answer 404 and Caddy answers 502 for a
// workload that is not up yet, and both are deployed correctly. An app with no domains is skipped.
func (i *Ingress) verify(ctx context.Context, app *storage.AppConfig) error {
	unanswered := make([]string, 0, len(app.Domains))
	var cause error
	for _, domain := range app.Domains {
		err := i.probeDomain(ctx, domain)
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
			app.Name, strings.Join(unanswered, ", "), cause)
	}
	return nil
}
