package engine

import (
	"fmt"
	"io"

	"github.com/FacileStudio/gare/apps/api/internal/deploys"
)

// DeployOptions says who started a deploy and, for a deploy the webhook queue recorded while it
// waited for its turn, which queued record it runs under.
type DeployOptions struct {
	Trigger  deploys.Trigger
	QueuedID string
}

// logSink passes everything to the caller's sink unchanged and copies what concerns the app
// being deployed into its deploy log, so the terminal looks the same and the history holds it too.
type logSink struct {
	next Sink
	app  string
	rec  *deploys.Recorder
}

// Event forwards one progress line and logs it when it is about the deploying app.
func (s logSink) Event(app string, kind Kind, msg string) {
	s.next.Event(app, kind, msg)
	if app == s.app {
		s.rec.Line(logLine(kind, msg))
	}
}

// Output tees the deploying app's subprocess output into its log.
func (s logSink) Output(app string) (io.Writer, io.Writer) {
	stdout, stderr := s.next.Output(app)
	if app != s.app {
		return stdout, stderr
	}
	return io.MultiWriter(stdout, s.rec), io.MultiWriter(stderr, s.rec)
}

// logLine marks a progress line with the icon the terminal shows for its kind, so a log reads like
// the deploy did on screen.
func logLine(kind Kind, msg string) string {
	icons := map[Kind]string{KindStep: "▸", KindDone: "✓", KindWarn: "!", KindDetail: "·"}
	return icons[kind] + " " + msg
}

// History returns the deploy history of an existing app, for a caller that lists or follows it.
func (e *Engine) History(name string) (deploys.History, error) {
	if _, err := e.Config(name); err != nil {
		return deploys.History{}, err
	}
	return deploys.Open(e.store.AppDir(name), name), nil
}

// QueueDeploy records a deploy that waits for its turn, so it shows in the history before it runs.
// The caller later runs it with Deploy and the returned id as DeployOptions.QueuedID.
func (e *Engine) QueueDeploy(name string, trigger deploys.Trigger) (deploys.Deploy, error) {
	history, err := e.History(name)
	if err != nil {
		return deploys.Deploy{}, err
	}
	return history.Queue(deploys.Request{Trigger: trigger})
}

// startRecord creates the record a deploy runs under, or resumes the one it was queued with, and
// returns an engine that reports through the recording sink.
func (e *Engine) startRecord(name string, req deploys.Request, queuedID string) (*Engine, *deploys.Recorder, error) {
	history, err := e.History(name)
	if err != nil {
		return nil, nil, err
	}
	var rec *deploys.Recorder
	if queuedID != "" {
		rec, err = history.Resume(queuedID)
	} else {
		rec, err = history.Start(req)
	}
	if err != nil {
		return nil, nil, fmt.Errorf("failed to record the deploy of %s: %w", name, err)
	}
	recording := *e
	recording.sink = logSink{next: e.sink, app: name, rec: rec}
	return &recording, rec, nil
}

// finishRecord ends a deploy's log with how it went and settles its record. A record that cannot
// be written is a warning: the deploy itself already happened, and its outcome is what the caller
// must hear.
func (e *Engine) finishRecord(name string, rec *deploys.Recorder, cause error) {
	if cause != nil {
		rec.Line("✗ " + cause.Error())
	}
	if err := rec.Finish(cause); err != nil {
		e.report(name).Warn(fmt.Sprintf("Could not record deploy %s (%v)", rec.Record().ID, err))
	}
}
