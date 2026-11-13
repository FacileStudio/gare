package server

import (
	"context"
	"sync"
	"time"
)

// DefaultMaxParallel caps concurrent deploys across apps when the caller sets no limit, so a burst
// of pushes to many apps cannot start more builds than a small host can carry.
const DefaultMaxParallel = 2

// deployTimeout bounds one deploy so a hung build cannot hold its slot forever.
const deployTimeout = 15 * time.Minute

// Answers a queued delivery reports, so the sender sees whether its push started, joined or
// followed a deploy.
const (
	answerQueued      = "queued"
	answerAlreadyNext = "already queued"
	answerRunningNext = "running, queued next"
)

// appState tracks one app's deploy worker: whether a worker exists, whether it holds a slot and
// runs a deploy now, and whether another deploy must follow the current one.
type appState struct {
	deploying bool
	pending   bool
}

// deployQueue coalesces deliveries into at most one running and one pending deploy per app, and
// bounds deploys across apps with a semaphore. A pending deploy needs no commit of its own because
// every deploy syncs to the branch head.
type deployQueue struct {
	run    func(ctx context.Context, app string) error
	sem    chan struct{}
	stop   chan struct{}
	mu     sync.Mutex
	apps   map[string]*appState
	closed bool
	wg     sync.WaitGroup
}

// newDeployQueue builds a queue that runs at most maxParallel deploys at once, defaulting to
// DefaultMaxParallel when the caller passes no positive limit.
func newDeployQueue(run func(ctx context.Context, app string) error, maxParallel int) *deployQueue {
	if maxParallel <= 0 {
		maxParallel = DefaultMaxParallel
	}
	return &deployQueue{
		run:  run,
		sem:  make(chan struct{}, maxParallel),
		stop: make(chan struct{}),
		apps: map[string]*appState{},
	}
}

// enqueue records a delivery for app and answers what became of it. It returns false once the
// queue is closed, because a stopping daemon must not start new work.
func (q *deployQueue) enqueue(app string) (string, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closed {
		return "", false
	}
	st, ok := q.apps[app]
	switch {
	case !ok:
		st = &appState{}
		q.apps[app] = st
		st.deploying = q.tryAcquire()
		holding := st.deploying
		q.wg.Go(func() { q.work(app, holding) })
		return answerQueued, true
	case !st.deploying || st.pending:
		return answerAlreadyNext, true
	default:
		st.pending = true
		return answerRunningNext, true
	}
}

// tryAcquire takes a deploy slot when one is free, so a delivery that finds room starts at once
// and counts as running from the moment it is answered.
func (q *deployQueue) tryAcquire() bool {
	select {
	case q.sem <- struct{}{}:
		return true
	default:
		return false
	}
}

// close stops the queue from accepting deliveries and wakes deploys waiting for a slot so they
// drop out; running deploys finish on their own.
func (q *deployQueue) close() {
	q.mu.Lock()
	defer q.mu.Unlock()
	if !q.closed {
		q.closed = true
		close(q.stop)
	}
}
