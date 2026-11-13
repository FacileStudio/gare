package apps

import (
	"context"

	"github.com/FacileStudio/gare/apps/api/internal/apikit"
	"github.com/FacileStudio/gare/apps/api/internal/deploys"
	"github.com/FacileStudio/gare/apps/api/internal/systemd"
	troncerrors "github.com/FacileStudio/tronc/errors"
)

// Service reads and changes apps through the engine.
type Service struct {
	deps       apikit.Deps
	controller controller
}

// NewService builds the module over the daemon's engine, store and queue.
func NewService(deps apikit.Deps) *Service {
	return &Service{deps: deps, controller: controller{allowLocal: deps.AllowLocalRepos}}
}

// list answers every app gare manages, in the store's order. An app whose configuration cannot be
// read is left out, as `gare list` leaves it out.
func (s *Service) list(ctx context.Context) (AppList, error) {
	cfgs, err := s.deps.Store.List()
	if err != nil {
		return AppList{}, troncerrors.Internal("the apps cannot be listed", err)
	}
	list := AppList{Apps: make([]App, 0, len(cfgs))}
	for _, cfg := range cfgs {
		app, err := s.view(ctx, cfg.Name)
		if err != nil {
			return AppList{}, err
		}
		list.Apps = append(list.Apps, app)
	}
	return list, nil
}

// view answers one app: its configuration and commit, its unit's state from systemd, its newest
// deploy and whether it can be rolled back now.
func (s *Service) view(ctx context.Context, name string) (App, error) {
	if _, err := s.deps.Known(name); err != nil {
		return App{}, err
	}
	st, err := s.deps.Engine.Status(ctx, name)
	if err != nil {
		return App{}, apikit.Failed(err)
	}
	return s.controller.app(st, s.lastDeploy(name), s.deps.Engine.CanRollback(ctx, name)), nil
}

// lastDeploy is the app's newest deploy, nil when it never deployed or its history is unreadable.
func (s *Service) lastDeploy(name string) *deploys.Deploy {
	history, err := s.deps.Engine.History(name)
	if err != nil {
		return nil
	}
	list, err := history.List()
	if err != nil || len(list) == 0 {
		return nil
	}
	return &list[0]
}

// streamLogs follows the app's journal into stream, starting lines back, until ctx ends: the client
// went away or the daemon stops. journalctl's own messages reach the stream too, and a journal that
// cannot be read at all ends with an error event, so the client says why instead of staying silent.
func (s *Service) streamLogs(ctx context.Context, name string, lines int, stream *apikit.Stream) {
	err := systemd.StreamLogs(ctx, name, systemd.LogOptions{
		Follow: true,
		Lines:  lines,
		Stdout: apikit.NewLineWriter(stream),
		Stderr: apikit.NewLineWriter(stream),
	})
	if err == nil || ctx.Err() != nil {
		return
	}
	if err := stream.Send(apikit.Event{Name: "error", Data: "the journal cannot be read: " + err.Error()}); err != nil {
		return
	}
}
