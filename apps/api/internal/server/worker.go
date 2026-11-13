package server

import (
	"context"
	"fmt"
	"os"

	"github.com/FacileStudio/gare/apps/api/internal/deploys"
	"github.com/FacileStudio/gare/apps/api/internal/engine"
)

// work runs app's deploys until nothing is pending, releasing its slot between deploys so other
// apps waiting for one get their turn.
func (q *deployQueue) work(app string, holding bool) {
	for {
		if !holding && !q.acquire(app) {
			return
		}
		q.deploy(app)
		<-q.sem
		holding = false
		if !q.next(app) {
			return
		}
	}
}

// acquire waits for a deploy slot and marks app as deploying. It gives up when the queue closes
// first, dropping the deploy that was still waiting.
func (q *deployQueue) acquire(app string) bool {
	select {
	case q.sem <- struct{}{}:
	case <-q.stop:
		q.drop(app)
		return false
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closed {
		<-q.sem
		q.forget(app)
		return false
	}
	q.apps[app].deploying = true
	return true
}

// next ends app's current deploy and reports whether a pending one should run. A pending deploy
// found after close is dropped rather than started.
func (q *deployQueue) next(app string) bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	st := q.apps[app]
	st.deploying = false
	if !st.pending {
		delete(q.apps, app)
		return false
	}
	st.pending, st.current, st.next = false, st.next, deploys.Deploy{}
	if q.closed {
		q.forget(app)
		return false
	}
	return true
}

// deploy runs one deploy of app under its timeout, as the queued record it waited under, and logs
// a failure, since no caller is left waiting for the result of an asynchronous deploy.
func (q *deployQueue) deploy(app string) {
	q.mu.Lock()
	record, deployer := q.apps[app].current, q.deployer
	q.apps[app].current = deploys.Deploy{}
	q.mu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), deployTimeout)
	defer cancel()
	opts := engine.DeployOptions{Trigger: record.Trigger, QueuedID: record.ID}
	if _, err := deployer.Deploy(ctx, app, opts); err != nil {
		fmt.Fprintf(os.Stderr, "deploy error for %s: %v\n", app, err)
	}
}

// drop forgets app's waiting deploy under the queue lock.
func (q *deployQueue) drop(app string) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.forget(app)
}

// forget removes app from the queue, settles the record of the deploy it drops and logs it, so an
// operator can see which pushes a shutdown left undeployed. The caller holds the lock.
func (q *deployQueue) forget(app string) {
	if st := q.apps[app]; st != nil {
		dropRecord(q.deployer, app, st.current)
	}
	delete(q.apps, app)
	fmt.Fprintf(os.Stderr, "dropped pending deploy for %s: server shutting down\n", app)
}
