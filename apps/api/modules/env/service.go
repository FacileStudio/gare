package env

import (
	"context"
	"fmt"

	"github.com/FacileStudio/gare/apps/api/internal/apikit"
	troncerrors "github.com/FacileStudio/tronc/errors"
)

// Service reads and changes an app's variables through the engine, in the store its workload reads.
type Service struct {
	deps apikit.Deps
}

// NewService builds the module over the daemon's engine and store.
func NewService(deps apikit.Deps) *Service {
	return &Service{deps: deps}
}

// vars reads a known app's variables.
func (s *Service) vars(name string) (map[string]string, error) {
	if _, err := s.deps.Known(name); err != nil {
		return nil, err
	}
	vars, err := s.deps.Engine.Env(name)
	if err != nil {
		return nil, apikit.Failed(err)
	}
	return vars, nil
}

// list answers the app's variables, masked.
func (s *Service) list(name string) (EnvList, error) {
	vars, err := s.vars(name)
	if err != nil {
		return EnvList{}, err
	}
	return entries(vars), nil
}

// reveal answers one variable in clear, 404 when the app has no such variable.
func (s *Service) reveal(name, key string) (Revealed, error) {
	vars, err := s.vars(name)
	if err != nil {
		return Revealed{}, err
	}
	value, ok := vars[key]
	if !ok {
		return Revealed{}, troncerrors.NotFound(fmt.Sprintf("%s has no variable %s", name, key))
	}
	return Revealed{Key: key, Value: value}, nil
}

// patch merges the change into the app's variables, then answers them masked. The engine restarts a
// running app so it reads them.
func (s *Service) patch(ctx context.Context, name string, change EnvPatch) (EnvList, error) {
	if _, err := s.deps.Known(name); err != nil {
		return EnvList{}, err
	}
	if err := checkPatch(change); err != nil {
		return EnvList{}, err
	}
	if err := s.deps.Guard(ctx, name); err != nil {
		return EnvList{}, err
	}
	if len(change.Set) > 0 {
		if _, err := s.deps.Engine.SetEnv(ctx, name, change.Set); err != nil {
			return EnvList{}, apikit.Failed(err)
		}
	}
	if len(change.Unset) > 0 {
		if _, err := s.deps.Engine.UnsetEnv(ctx, name, change.Unset); err != nil {
			return EnvList{}, apikit.Failed(err)
		}
	}
	return s.list(name)
}
