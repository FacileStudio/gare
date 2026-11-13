// Package workload holds everything gare does differently for a container, a static site and a
// compose stack, behind one interface selected once from an app's configuration, so no caller asks
// which type it holds again and a new type is one new adapter.
package workload

import (
	"context"
	"io"

	"github.com/FacileStudio/gare/apps/api/internal/storage"
)

// Stage tells an adapter whether it is settling or preparing an app being created or one being
// deployed, because a create writes the artifacts of a fresh checkout while a deploy also builds.
type Stage int

const (
	// Create is the first write of an app's artifacts, right after its repository is cloned.
	Create Stage = iota
	// Deploy is a rebuild of an existing app from its freshly synced repository.
	Deploy
)

// Reporter is where an adapter says what it is doing, so the engine decides how progress reaches
// the operator, a log or an API client while the adapter never prints.
type Reporter interface {
	Step(msg string)
	Done(msg string)
	Warn(msg string)
	Verbose(msg string)
}

// App is one application as an adapter acts on it: its configuration, its storage directory, the
// store it may allocate a port from, where it reports progress and where subprocess output goes.
type App struct {
	Config *storage.AppConfig
	Dir    string
	Store  *storage.Store
	Report Reporter
	Stdout io.Writer
	Stderr io.Writer
}

// RepoDir returns the app's git checkout.
func (a App) RepoDir() string {
	return storage.GetRepoDir(a.Dir)
}

// Builder is the part of a workload type that writes, builds and removes what the app runs from.
type Builder interface {
	Settle(app App, stage Stage) error
	Prepare(ctx context.Context, app App, stage Stage) error
	Teardown(ctx context.Context, app App, volumes bool)
	Env() EnvStore
}

// Runner is the part of a workload type that checks an already deployed app around a start and
// after a stop.
type Runner interface {
	CheckStart(ctx context.Context, app App) error
	CheckStopped(ctx context.Context, app App)
}

// Workload is one workload type: how it is built and how it is run.
type Workload interface {
	Builder
	Runner
}

// For returns the adapter of the workload type an app's configuration names. It refuses a type it
// does not know rather than defaulting to a container, because acting on a guessed type is how a
// teardown once left a compose stack running.
func For(cfg *storage.AppConfig) (Workload, error) {
	workloadType, err := cfg.ResolveWorkloadType()
	if err != nil {
		return nil, err
	}
	switch workloadType {
	case storage.WorkloadStatic:
		return static{}, nil
	case storage.WorkloadCompose:
		return composeStack{}, nil
	default:
		return container{}, nil
	}
}
