package server

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/FacileStudio/gare/apps/api/internal/deploys"
	"github.com/FacileStudio/gare/apps/api/internal/engine"
)

// Deployer is what the queue runs deploys through: the engine, which records a deploy as queued
// the moment it is accepted and runs it later under that record, so the history shows a waiting
// deploy and a client can follow it before it starts.
type Deployer interface {
	QueueDeploy(app string, trigger deploys.Trigger) (deploys.Deploy, error)
	Deploy(ctx context.Context, app string, opts engine.DeployOptions) (*engine.DeployResult, error)
	History(app string) (deploys.History, error)
}

// handlerDeployer runs Config.DeployHandler for a server no engine was attached to. It keeps no
// history, so its queued records are empty and nothing is settled when one is dropped.
type handlerDeployer struct {
	run func(ctx context.Context, app string) error
}

// QueueDeploy returns an unrecorded deploy that only carries its trigger.
func (h handlerDeployer) QueueDeploy(app string, trigger deploys.Trigger) (deploys.Deploy, error) {
	return deploys.Deploy{App: app, Trigger: trigger, Status: deploys.StatusQueued}, nil
}

// Deploy calls the handler, when there is one.
func (h handlerDeployer) Deploy(ctx context.Context, app string, _ engine.DeployOptions) (*engine.DeployResult, error) {
	if h.run == nil {
		return nil, nil
	}
	return nil, h.run(ctx, app)
}

// History has nothing to return: a handler keeps no records.
func (h handlerDeployer) History(string) (deploys.History, error) {
	return deploys.History{}, errors.New("no deploy history without an engine")
}

// UseDeployer makes every deploy the queue runs from now on, webhook and API alike, go through
// deployer instead of Config.DeployHandler. The daemon calls it with its engine before it serves.
func (s *Server) UseDeployer(deployer Deployer) {
	s.queue.mu.Lock()
	defer s.queue.mu.Unlock()
	s.queue.deployer = deployer
}

// Enqueue asks for a deploy of app on behalf of trigger and returns the queued record it runs
// under. A deploy already waiting for app absorbs the request and its record is returned, so API
// and webhook requests for one app coalesce and share the parallelism limit. It fails with
// ErrClosed while the server shuts down.
func (s *Server) Enqueue(app string, trigger deploys.Trigger) (deploys.Deploy, error) {
	t, err := s.queue.enqueue(app, trigger)
	return t.record, err
}

// Go runs fn in the background on a context of its own, bounded like a deploy, and makes Shutdown
// wait for it as it waits for running deploys. It fails with ErrClosed once shutdown began.
func (s *Server) Go(fn func(ctx context.Context)) error {
	q := s.queue
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closed {
		return ErrClosed
	}
	q.wg.Go(func() {
		ctx, cancel := context.WithTimeout(context.Background(), deployTimeout)
		defer cancel()
		fn(ctx)
	})
	return nil
}

// dropRecord settles the queued record of a deploy a shutdown drops as failed, so the history
// never shows a deploy waiting for a turn that will not come.
func dropRecord(deployer Deployer, app string, record deploys.Deploy) {
	if record.ID == "" {
		return
	}
	history, err := deployer.History(app)
	if err != nil {
		return
	}
	rec, err := history.Resume(record.ID)
	if err == nil {
		err = rec.Finish(errors.New("dropped: the server shut down before this deploy ran"))
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not settle dropped deploy %s of %s: %v\n", record.ID, app, err)
	}
}
