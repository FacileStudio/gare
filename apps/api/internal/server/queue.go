package server

import (
	"errors"
	"sync"
	"time"

	"github.com/FacileStudio/gare/apps/api/internal/deploys"
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

// ErrClosed is what Enqueue answers once the server is shutting down and takes no new deploy.
var ErrClosed = errors.New("the server is shutting down")

// appState tracks one app's deploy worker: whether a worker exists, whether it holds a slot and
// runs a deploy now, and whether another deploy must follow the current one. current is the queued
// record of the deploy the worker runs next, until it starts; next is the record of the pending one,
// which every delivery coalesced into it shares.
type appState struct {
	deploying bool
	pending   bool
	current   deploys.Deploy
	next      deploys.Deploy
}

// ticket is what became of one delivery: the answer a forge reads and the queued record an API
// client follows.
type ticket struct {
	answer string
	record deploys.Deploy
}

// deployQueue coalesces deliveries into at most one running and one pending deploy per app, and
// bounds deploys across apps with a semaphore. A pending deploy needs no commit of its own because
// every deploy syncs to the branch head.
type deployQueue struct {
	deployer Deployer
	sem      chan struct{}
	stop     chan struct{}
	mu       sync.Mutex
	apps     map[string]*appState
	closed   bool
	wg       sync.WaitGroup
}

// newDeployQueue builds a queue that runs at most maxParallel deploys at once, defaulting to
// DefaultMaxParallel when the caller passes no positive limit.
func newDeployQueue(deployer Deployer, maxParallel int) *deployQueue {
	if maxParallel <= 0 {
		maxParallel = DefaultMaxParallel
	}
	return &deployQueue{
		deployer: deployer,
		sem:      make(chan struct{}, maxParallel),
		stop:     make(chan struct{}),
		apps:     map[string]*appState{},
	}
}

// enqueue records a delivery for app and says what became of it. A delivery that starts or follows
// a deploy records a queued deploy for it; one that joins a deploy already waiting shares that
// deploy's record. It fails with ErrClosed once the queue is closed, because a stopping daemon must
// not start new work.
func (q *deployQueue) enqueue(app string, trigger deploys.Trigger) (ticket, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closed {
		return ticket{}, ErrClosed
	}
	st, ok := q.apps[app]
	switch {
	case ok && !st.deploying:
		return ticket{answer: answerAlreadyNext, record: st.current}, nil
	case ok && st.pending:
		return ticket{answer: answerAlreadyNext, record: st.next}, nil
	}
	record, err := q.deployer.QueueDeploy(app, trigger)
	if err != nil {
		return ticket{}, err
	}
	if ok {
		st.pending, st.next = true, record
		return ticket{answer: answerRunningNext, record: record}, nil
	}
	st = &appState{current: record, deploying: q.tryAcquire()}
	q.apps[app] = st
	holding := st.deploying
	q.wg.Go(func() { q.work(app, holding) })
	return ticket{answer: answerQueued, record: record}, nil
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
