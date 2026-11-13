package deploys

import (
	"errors"
	"time"
)

// Finish closes the log and records the deploy as succeeded when cause is nil and failed otherwise.
func (r *Recorder) Finish(cause error) error {
	if cause == nil {
		return r.finish(StatusSucceeded, nil, "")
	}
	return r.finish(StatusFailed, cause, "")
}

// FinishRolledBack closes the log and records a deploy that failed its health probes and was
// replaced by deploy to.
func (r *Recorder) FinishRolledBack(cause error, to string) error {
	return r.finish(StatusRolledBack, cause, to)
}

// finish ends the log on a full line, syncs and closes it, then writes the final record, in that
// order, so a reader that sees a final status knows the log is complete.
func (r *Recorder) finish(status Status, cause error, rolledBackTo string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.atLineStart {
		r.append([]byte("\n"))
	}
	logErr := errors.Join(r.err, r.log.Sync(), r.log.Close())
	now := time.Now().UTC()
	r.rec.Status, r.rec.FinishedAt = status, &now
	if cause != nil {
		msg := cause.Error()
		r.rec.Error = &msg
	}
	if rolledBackTo != "" {
		r.rec.RolledBackTo = &rolledBackTo
	}
	return errors.Join(logErr, r.save())
}
