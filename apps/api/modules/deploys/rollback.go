package deploys

import (
	"context"
	"errors"
	"time"

	"github.com/FacileStudio/gare/apps/api/internal/deploys"
	"github.com/FacileStudio/gare/apps/api/internal/engine"
	"github.com/FacileStudio/gare/apps/api/internal/server"
	troncerrors "github.com/FacileStudio/tronc/errors"
)

// rollbackStartWait bounds how long the route waits for a rollback it started to record itself. The
// engine records it once it holds the lock and has found its target, which takes a moment, not a
// deploy.
const rollbackStartWait = 10 * time.Second

// rollbackPoll is how often the route looks for that record.
const rollbackPoll = 50 * time.Millisecond

// rollback starts a rollback of the app in the background and answers its record as soon as the
// engine wrote it. A rollback the engine refuses before it records anything, such as one of a
// static app or one with no earlier image, is answered with the engine's reason.
func (s *Service) rollback(ctx context.Context, name, deployID string) (Deploy, error) {
	history, err := s.history(name)
	if err != nil {
		return Deploy{}, err
	}
	if deployID != "" {
		if _, err := record(history, name, deployID); err != nil {
			return Deploy{}, err
		}
	}
	if err := s.deps.Guard(ctx, name); err != nil {
		return Deploy{}, err
	}
	before, err := history.List()
	if err != nil {
		return Deploy{}, troncerrors.Internal("the deploy history cannot be read", err)
	}
	done := make(chan error, 1)
	opts := engine.RollbackOptions{Trigger: deploys.TriggerAPI, To: deployID}
	err = s.deps.Queue.Go(func(ctx context.Context) {
		_, err := s.deps.Engine.Rollback(ctx, name, opts)
		done <- err
	})
	if errors.Is(err, server.ErrClosed) {
		return Deploy{}, troncerrors.Unavailable("gare is restarting; roll back again in a moment")
	}
	return awaitRollback(history, ids(before), done)
}

// awaitRollback waits for the record of the rollback that is running, or for the rollback to end
// without one.
func awaitRollback(history deploys.History, before map[string]bool, done <-chan error) (Deploy, error) {
	deadline := time.After(rollbackStartWait)
	for {
		if rec, ok := newRollback(history, before); ok {
			return rec, nil
		}
		select {
		case err := <-done:
			return endedRollback(history, before, err)
		case <-deadline:
			return Deploy{}, troncerrors.Internal("the rollback did not start in time", nil)
		case <-time.After(rollbackPoll):
		}
	}
}

// endedRollback answers a rollback that ended before its record was seen: its record when it wrote
// one after all, else the engine's reason for refusing it.
func endedRollback(history deploys.History, before map[string]bool, err error) (Deploy, error) {
	if rec, ok := newRollback(history, before); ok {
		return rec, nil
	}
	if err != nil {
		return Deploy{}, troncerrors.Failed(err.Error())
	}
	return Deploy{}, troncerrors.Internal("the rollback finished without a record", nil)
}

// newRollback finds the record an API rollback wrote since before was read.
func newRollback(history deploys.History, before map[string]bool) (Deploy, bool) {
	list, err := history.List()
	if err != nil {
		return Deploy{}, false
	}
	for _, d := range list {
		if !before[d.ID] && d.Trigger == deploys.TriggerAPI && d.RolledBackTo != nil {
			return d, true
		}
	}
	return Deploy{}, false
}
