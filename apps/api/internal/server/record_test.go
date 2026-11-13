package server

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/FacileStudio/gare/apps/api/internal/deploys"
	"github.com/FacileStudio/gare/apps/api/internal/engine"
	"github.com/FacileStudio/gare/apps/api/internal/storage"
)

// quietSink drops what the engine reports.
type quietSink struct{}

// Event drops a progress line.
func (quietSink) Event(string, engine.Kind, string) {}

// Output discards subprocess output.
func (quietSink) Output(string) (io.Writer, io.Writer) { return io.Discard, io.Discard }

// gatedEngine is a real engine whose deploys wait for release, so a test can pile requests up
// behind a running deploy.
type gatedEngine struct {
	*engine.Engine
	release chan struct{}
	runs    atomic.Int32
}

// Deploy waits for release, then runs the real deploy under its queued record.
func (g *gatedEngine) Deploy(ctx context.Context, app string, opts engine.DeployOptions) (*engine.DeployResult, error) {
	g.runs.Add(1)
	<-g.release
	return g.Engine.Deploy(ctx, app, opts)
}

// newRecordingServer returns a server whose queue records deploys of app "app" in a temp store.
func newRecordingServer(t *testing.T) (*Server, *gatedEngine) {
	t.Helper()
	store := storage.Open(t.TempDir())
	if err := store.Save(&storage.AppConfig{Name: "app", Branch: "main"}); err != nil {
		t.Fatal(err)
	}
	g := &gatedEngine{Engine: engine.New(store, quietSink{}, engine.Settings{}), release: make(chan struct{})}
	s := newSignedServer(nil)
	s.UseDeployer(g)
	return s, g
}

// TestWebhookAndAPIDeploysCoalesce proves one queue serves both: a push starts a deploy, an API
// request queues the next one with its own record, and later requests from either side join that
// record instead of adding a third deploy.
func TestWebhookAndAPIDeploysCoalesce(t *testing.T) {
	s, g := newRecordingServer(t)
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()
	if status, body := postAnswer(t, ts.URL+"/webhook/app"); status != http.StatusAccepted || body != answerQueued {
		t.Fatalf("first push = %d %q", status, body)
	}
	next, err := s.Enqueue("app", deploys.TriggerAPI)
	if err != nil || next.Status != deploys.StatusQueued || next.Trigger != deploys.TriggerAPI || next.ID == "" {
		t.Fatalf("API deploy = %+v, %v; want a queued api record", next, err)
	}
	if _, body := postAnswer(t, ts.URL+"/webhook/app"); body != answerAlreadyNext {
		t.Fatalf("second push = %q, want %q", body, answerAlreadyNext)
	}
	if again, _ := s.Enqueue("app", deploys.TriggerAPI); again.ID != next.ID {
		t.Fatalf("second API deploy got record %s, want the waiting %s", again.ID, next.ID)
	}
	close(g.release)
	waitQueueIdle(t, s)
	assertHistory(t, g, []deploys.Trigger{deploys.TriggerAPI, deploys.TriggerWebhook})
	if g.runs.Load() != 2 {
		t.Fatalf("ran %d deploys, want 2", g.runs.Load())
	}
}

// TestShutdownSettlesTheDroppedRecord proves a deploy a shutdown drops is not left queued forever.
func TestShutdownSettlesTheDroppedRecord(t *testing.T) {
	s, g := newRecordingServer(t)
	if _, err := s.Enqueue("app", deploys.TriggerAPI); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Enqueue("app", deploys.TriggerAPI); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- s.Shutdown(context.Background()) }()
	waitQueueClosed(s)
	close(g.release)
	if err := <-done; err != nil {
		t.Fatalf("shutdown: %v", err)
	}
	history, _ := g.History("app")
	list, _ := history.List()
	for _, d := range list {
		if !d.Finished() {
			t.Fatalf("record %+v is still waiting after shutdown", d)
		}
	}
	if _, err := s.Enqueue("app", deploys.TriggerAPI); err != ErrClosed {
		t.Fatalf("enqueue after shutdown = %v, want ErrClosed", err)
	}
}

// assertHistory checks the app's records, newest first, have the given triggers and all finished.
func assertHistory(t *testing.T, g *gatedEngine, triggers []deploys.Trigger) {
	t.Helper()
	history, err := g.History("app")
	if err != nil {
		t.Fatal(err)
	}
	list, err := history.List()
	if err != nil || len(list) != len(triggers) {
		t.Fatalf("history = %+v, %v; want %d records", list, err, len(triggers))
	}
	for i, d := range list {
		if d.Trigger != triggers[i] || !d.Finished() {
			t.Errorf("record %d = %+v, want a finished %s deploy", i, d, triggers[i])
		}
	}
}
