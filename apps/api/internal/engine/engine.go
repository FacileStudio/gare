// Package engine owns an application's lifecycle: create, deploy, start, stop, restart, destroy,
// status and the env, domain and tag changes. It prints nothing and reads no global state: progress
// goes to the Sink its caller supplies and every method returns a typed result, so the CLI, the
// webhook daemon and the HTTP API drive the same code and only render it differently.
package engine

import (
	"io"

	"github.com/FacileStudio/gare/apps/api/internal/git"
	"github.com/FacileStudio/gare/apps/api/internal/notify"
	"github.com/FacileStudio/gare/apps/api/internal/storage"
	"github.com/FacileStudio/gare/apps/api/internal/workload"
)

// Kind tells a sink what a progress line is, so each caller can render it in its own way.
type Kind int

const (
	// KindStep announces a step the engine is about to run.
	KindStep Kind = iota
	// KindDone reports a step that finished while the operation goes on.
	KindDone
	// KindWarn reports something the operator should know about that does not stop the operation.
	KindWarn
	// KindDetail is an extra line only emitted when the settings ask for verbose output.
	KindDetail
)

// Sink receives everything the engine reports about one app while it works. Output returns the
// writers a long-running subprocess, such as git or podman build, streams to.
type Sink interface {
	Event(app string, kind Kind, msg string)
	Output(app string) (stdout, stderr io.Writer)
}

// Settings is the configuration one caller runs the engine with: whether to emit detail lines and
// how git authenticates. The CLI resolves it from ~/.gare.yml and its flags; the daemon passes its
// own.
type Settings struct {
	Verbose bool
	Auth    git.Auth
}

// Engine runs application lifecycle operations against one store.
type Engine struct {
	store    *storage.Store
	sink     Sink
	settings Settings
	alerts   *notify.Notifier
}

// New returns an engine acting on the apps in store, reporting to sink. It reads the Antenne alert
// settings from the environment once, here, and sends no alert when they are unset.
func New(store *storage.Store, sink Sink, settings Settings) *Engine {
	return &Engine{store: store, sink: sink, settings: settings, alerts: notify.FromEnv()}
}

// reporter is the sink narrowed to one app, in the shape a workload adapter reports through.
type reporter struct {
	sink    Sink
	app     string
	verbose bool
}

// Step announces a step.
func (r reporter) Step(msg string) { r.sink.Event(r.app, KindStep, msg) }

// Done reports a finished step.
func (r reporter) Done(msg string) { r.sink.Event(r.app, KindDone, msg) }

// Warn reports a warning.
func (r reporter) Warn(msg string) { r.sink.Event(r.app, KindWarn, msg) }

// Verbose reports a detail line, dropped unless the settings ask for verbose output.
func (r reporter) Verbose(msg string) {
	if r.verbose {
		r.sink.Event(r.app, KindDetail, msg)
	}
}

// report returns the reporter for one app.
func (e *Engine) report(app string) reporter {
	return reporter{sink: e.sink, app: app, verbose: e.settings.Verbose}
}

// workloadApp returns an app in the shape a workload adapter acts on.
func (e *Engine) workloadApp(cfg *storage.AppConfig) workload.App {
	stdout, stderr := e.sink.Output(cfg.Name)
	return workload.App{
		Config: cfg,
		Dir:    e.store.AppDir(cfg.Name),
		Store:  e.store,
		Report: e.report(cfg.Name),
		Stdout: stdout,
		Stderr: stderr,
	}
}
