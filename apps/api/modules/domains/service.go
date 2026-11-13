package domains

import (
	"context"
	"fmt"

	"github.com/FacileStudio/gare/apps/api/internal/apikit"
	troncerrors "github.com/FacileStudio/tronc/errors"
)

// Service reads and changes an app's hostnames through the engine.
type Service struct {
	deps apikit.Deps
}

// NewService builds the module over the daemon's engine and store.
func NewService(deps apikit.Deps) *Service {
	return &Service{deps: deps}
}

// list answers the app's hostnames.
func (s *Service) list(name string) (DomainList, error) {
	cfg, err := s.deps.Known(name)
	if err != nil {
		return DomainList{}, err
	}
	return domainList(cfg), nil
}

// add routes a hostname to the app. A hostname any app already owns, this one included, is a 409.
func (s *Service) add(ctx context.Context, name, hostname string) (DomainList, error) {
	if _, err := s.deps.Known(name); err != nil {
		return DomainList{}, err
	}
	if err := checkHostname(hostname); err != nil {
		return DomainList{}, err
	}
	if err := s.deps.Guard(ctx, name); err != nil {
		return DomainList{}, err
	}
	if err := s.deps.Engine.AddDomain(ctx, name, hostname); err != nil {
		return DomainList{}, troncerrors.Conflict(err.Error())
	}
	return s.list(name)
}

// remove stops routing a hostname to the app, 404 when the app does not have it.
func (s *Service) remove(ctx context.Context, name, hostname string) (DomainList, error) {
	cfg, err := s.deps.Known(name)
	if err != nil {
		return DomainList{}, err
	}
	if !cfg.HasDomain(hostname) {
		return DomainList{}, troncerrors.NotFound(fmt.Sprintf("%s is not routed on %s", name, hostname))
	}
	if err := s.deps.Guard(ctx, name); err != nil {
		return DomainList{}, err
	}
	if err := s.deps.Engine.RemoveDomain(ctx, name, hostname); err != nil {
		return DomainList{}, apikit.Failed(err)
	}
	return s.list(name)
}
