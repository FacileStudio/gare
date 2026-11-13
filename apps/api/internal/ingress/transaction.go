package ingress

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/FacileStudio/gare/apps/api/internal/atomicfile"
	"github.com/FacileStudio/gare/apps/api/internal/storage"
)

// Reloader asks the running Caddy to load the configuration on disk. It is a field rather than a
// direct call so a test can refuse a reload without a Caddy on the machine.
type Reloader func(ctx context.Context) error

// Prober asks the ingress whether it answers for one hostname on one listener.
type Prober func(ctx context.Context, target Target) error

// Ingress owns one snippet directory and every step that changes what Caddy serves from it: write,
// reload, restore and probe. Keeping them together is what lets a refused reload be undone.
type Ingress struct {
	ConfDir string
	Reload  Reloader
	Probe   Prober
}

// snapshot is a snippet's state before a change: its exact bytes, or its absence.
type snapshot struct {
	path    string
	content []byte
	existed bool
}

// Default returns the ingress of this host: the configured snippet directory, a real Caddy reload
// and a loopback probe.
func Default() *Ingress {
	return &Ingress{ConfDir: ResolveConfDir(), Reload: Reload, Probe: ProbeTarget}
}

// Apply makes this host's Caddy serve the app as configured. See Ingress.Apply.
func Apply(ctx context.Context, app *storage.AppConfig) error {
	return Default().Apply(ctx, app)
}

// Remove takes the app out of this host's Caddy. See Ingress.Remove.
func Remove(ctx context.Context, name string) error {
	return Default().Remove(ctx, name)
}

// commit applies one change to an app's snippet as a transaction. When Caddy refuses the result,
// the previous snippet is put back byte for byte, or removed when there was none, and Caddy is
// reloaded again: a refused snippet left on disk would make every later reload by any app fail.
func (i *Ingress) commit(ctx context.Context, name string, change func() error) error {
	before, err := capture(GetSnippetPath(i.ConfDir, name))
	if err != nil {
		return fmt.Errorf("could not read the Caddy snippet for %s: %w", name, err)
	}
	if err := change(); err != nil {
		return fmt.Errorf("could not sync the Caddy snippet for %s: %w", name, err)
	}
	refused := i.Reload(ctx)
	if refused == nil {
		return nil
	}
	if err := before.restore(); err != nil {
		return fmt.Errorf("caddy did not accept the configuration for %s and its previous snippet could not be restored (%v): %w",
			name, err, refused)
	}
	if err := i.Reload(ctx); err != nil {
		return fmt.Errorf("caddy did not accept the configuration for %s; the change was rolled back but caddy refused the "+
			"restored configuration too (%v): %w", name, err, refused)
	}
	return fmt.Errorf("caddy did not accept the configuration for %s, so the change was rolled back: %w", name, refused)
}

// capture records a snippet's state so a refused change can be undone.
func capture(path string) (snapshot, error) {
	content, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return snapshot{path: path}, nil
	}
	if err != nil {
		return snapshot{}, err
	}
	return snapshot{path: path, content: content, existed: true}, nil
}

// restore puts the snippet back the way capture found it.
func (s snapshot) restore() error {
	if s.existed {
		return atomicfile.WriteFile(s.path, s.content, 0644)
	}
	if err := os.Remove(s.path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
