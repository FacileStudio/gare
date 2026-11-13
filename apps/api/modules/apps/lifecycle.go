package apps

import (
	"context"
	"fmt"

	"github.com/FacileStudio/gare/apps/api/internal/apikit"
	troncerrors "github.com/FacileStudio/tronc/errors"
)

// Lifecycle actions the routes take, by the name the path spells.
const (
	actionStart   = "start"
	actionStop    = "stop"
	actionRestart = "restart"
)

// create clones and sets up a new app through the engine, then answers it.
func (s *Service) create(ctx context.Context, body CreateRequest) (App, error) {
	req, err := s.controller.createRequest(body)
	if err != nil {
		return App{}, err
	}
	if _, err := s.deps.Engine.Config(req.Name); err == nil {
		return App{}, troncerrors.Conflict(fmt.Sprintf("an app named %s already exists", req.Name))
	}
	if err := s.deps.Guard(ctx, req.Name); err != nil {
		return App{}, err
	}
	if _, err := s.deps.Engine.Create(ctx, req); err != nil {
		return App{}, apikit.Rejected(err)
	}
	return s.view(ctx, req.Name)
}

// destroy tears an app down once the caller typed its name back, dropping compose volumes only
// when asked.
func (s *Service) destroy(ctx context.Context, name, confirm string, volumes bool) error {
	if _, err := s.deps.Known(name); err != nil {
		return err
	}
	if confirm != name {
		return troncerrors.Invalid(fmt.Sprintf("type %s in confirm to destroy it", name))
	}
	if err := s.deps.Guard(ctx, name); err != nil {
		return err
	}
	if err := s.deps.Engine.Destroy(ctx, name, volumes); err != nil {
		return apikit.Failed(err)
	}
	return nil
}

// lifecycle starts, stops or restarts an app's unit and answers the app as it then stands.
func (s *Service) lifecycle(ctx context.Context, name, action string) (App, error) {
	if _, err := s.deps.Known(name); err != nil {
		return App{}, err
	}
	if err := s.deps.Guard(ctx, name); err != nil {
		return App{}, err
	}
	var err error
	switch action {
	case actionStart:
		err = s.deps.Engine.Start(ctx, name)
	case actionStop:
		_, err = s.deps.Engine.Stop(ctx, name)
	case actionRestart:
		err = s.deps.Engine.Restart(ctx, name)
	}
	if err != nil {
		return App{}, apikit.Failed(err)
	}
	return s.view(ctx, name)
}

// setTags makes the app's tags exactly wanted, through the engine's add and remove.
func (s *Service) setTags(ctx context.Context, name string, wanted []string) (App, error) {
	cfg, err := s.deps.Known(name)
	if err != nil {
		return App{}, err
	}
	add, remove, err := tagChanges(cfg.Tags, wanted)
	if err != nil {
		return App{}, err
	}
	if err := s.deps.Guard(ctx, name); err != nil {
		return App{}, err
	}
	if _, _, err := s.deps.Engine.RemoveTags(ctx, name, remove); err != nil {
		return App{}, apikit.Failed(err)
	}
	if _, err := s.deps.Engine.AddTags(ctx, name, add); err != nil {
		return App{}, apikit.Rejected(err)
	}
	return s.view(ctx, name)
}
