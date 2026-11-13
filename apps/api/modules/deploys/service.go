package deploys

import (
	"errors"
	"fmt"

	"github.com/FacileStudio/gare/apps/api/internal/apikit"
	"github.com/FacileStudio/gare/apps/api/internal/deploys"
	"github.com/FacileStudio/gare/apps/api/internal/server"
	troncerrors "github.com/FacileStudio/tronc/errors"
)

// Service starts deploys through the queue and reads the history through the engine.
type Service struct {
	deps apikit.Deps
}

// NewService builds the module over the daemon's engine, store and queue.
func NewService(deps apikit.Deps) *Service {
	return &Service{deps: deps}
}

// deploy queues a deploy of the app and answers its queued record. A deploy already waiting for the
// app absorbs this one, and its record is the answer.
func (s *Service) deploy(name string) (Deploy, error) {
	if _, err := s.deps.Known(name); err != nil {
		return Deploy{}, err
	}
	record, err := s.deps.Queue.Enqueue(name, deploys.TriggerAPI)
	if errors.Is(err, server.ErrClosed) {
		return Deploy{}, troncerrors.Unavailable("gare is restarting; deploy again in a moment")
	}
	if err != nil {
		return Deploy{}, apikit.Failed(err)
	}
	return record, nil
}

// history opens the deploy history of a known app.
func (s *Service) history(name string) (deploys.History, error) {
	if _, err := s.deps.Known(name); err != nil {
		return deploys.History{}, err
	}
	history, err := s.deps.Engine.History(name)
	if err != nil {
		return deploys.History{}, apikit.Failed(err)
	}
	return history, nil
}

// list answers the app's deploys, newest first.
func (s *Service) list(name string) (DeployList, error) {
	history, err := s.history(name)
	if err != nil {
		return DeployList{}, err
	}
	list, err := history.List()
	if err != nil {
		return DeployList{}, troncerrors.Internal("the deploy history cannot be read", err)
	}
	return DeployList{Deploys: list}, nil
}

// get answers one deploy of the app, 404 for an id it has no record of.
func (s *Service) get(name, id string) (Deploy, error) {
	history, err := s.history(name)
	if err != nil {
		return Deploy{}, err
	}
	return record(history, name, id)
}

// record reads one deploy, turning an id that is malformed or unknown into a 404.
func record(history deploys.History, name, id string) (Deploy, error) {
	rec, err := history.Get(id)
	if deploys.ValidateID(id) != nil || errors.Is(err, deploys.ErrNotFound) {
		return Deploy{}, troncerrors.NotFound(fmt.Sprintf("%s has no deploy %q", name, id))
	}
	if err != nil {
		return Deploy{}, troncerrors.Internal("the deploy record cannot be read", err)
	}
	return rec, nil
}
