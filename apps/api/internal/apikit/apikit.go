// Package apikit holds what every HTTP module of the daemon shares: the engine and the queue they
// drive, the lookup that turns an unknown app into a 404, the short lock probe that turns a busy
// app into a 409, and the error shapes the tronc envelope carries to people.
package apikit

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"time"

	"github.com/FacileStudio/gare/apps/api/internal/appname"
	"github.com/FacileStudio/gare/apps/api/internal/deploys"
	"github.com/FacileStudio/gare/apps/api/internal/engine"
	"github.com/FacileStudio/gare/apps/api/internal/storage"
	troncerrors "github.com/FacileStudio/tronc/errors"
	"github.com/FacileStudio/tronc/httpjson"
	"github.com/go-chi/chi/v5"
)

// LockWait is how long a request waits for an app another operation holds before it answers 409,
// so a busy app gets a quick answer instead of a request that hangs for a whole deploy.
const LockWait = 2 * time.Second

// OperationTimeout bounds a synchronous operation a request starts, such as a create or a stop. It
// runs detached from the request, so a client that goes away cannot cut a clone or a stop in half.
const OperationTimeout = 10 * time.Minute

// Queue runs deploys and other long operations in the background on the daemon's context, and
// makes shutdown wait for them. The webhook intake is the one implementation.
type Queue interface {
	Enqueue(app string, trigger deploys.Trigger) (deploys.Deploy, error)
	Go(fn func(ctx context.Context)) error
}

// Deps is what a module is built over. AllowLocalRepos lets a create clone from a local absolute
// path, which only tests turn on: in production a repository is always a remote URL.
type Deps struct {
	Engine          *engine.Engine
	Store           *storage.Store
	Queue           Queue
	AllowLocalRepos bool
}

// Name is the app named in the request path.
func Name(r *http.Request) string {
	return chi.URLParam(r, "name")
}

// Known loads an app's configuration, answering 404 for a name gare does not manage, an invalid
// one included, since no such app can exist.
func (d Deps) Known(name string) (*storage.AppConfig, error) {
	if appname.Validate(name) != nil {
		return nil, troncerrors.NotFound(fmt.Sprintf("there is no app named %q", name))
	}
	cfg, err := d.Engine.Config(name)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, troncerrors.NotFound(fmt.Sprintf("there is no app named %q", name))
	}
	if err != nil {
		return nil, troncerrors.Internal(fmt.Sprintf("the configuration of %s cannot be read", name), err)
	}
	return cfg, nil
}

// Guard waits up to LockWait for the app's lock and releases it at once, answering 409 when another
// operation still holds it. The engine takes the lock again for the operation itself.
func (d Deps) Guard(ctx context.Context, name string) error {
	probe, cancel := context.WithTimeout(ctx, LockWait)
	defer cancel()
	unlock, err := d.Store.Lock(probe, name)
	if errors.Is(err, context.DeadlineExceeded) {
		return troncerrors.Conflict(fmt.Sprintf("another operation on %s is running; try again once it finishes", name))
	}
	if err != nil {
		return troncerrors.Internal(fmt.Sprintf("the lock of %s cannot be taken", name), err)
	}
	unlock()
	return nil
}

// Detach returns a context for an operation a request starts, which outlives the request but not
// OperationTimeout.
func Detach(r *http.Request) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.WithoutCancel(r.Context()), OperationTimeout)
}

// Respond writes value as JSON with status, or err in the tronc envelope when there is one.
func Respond(w http.ResponseWriter, status int, value any, err error) {
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	httpjson.WriteJSON(w, status, value)
}

// Failed is an operation that did not succeed, as a 500 whose message is the engine's own, written
// for the operator. An error that already has a status keeps it.
func Failed(err error) error {
	var shaped *troncerrors.Error
	if errors.As(err, &shaped) {
		return err
	}
	return troncerrors.Internal(err.Error(), err)
}

// Rejected is a request the engine refused, as a 400 carrying the engine's reason. An error that
// already has a status keeps it.
func Rejected(err error) error {
	var shaped *troncerrors.Error
	if errors.As(err, &shaped) {
		return err
	}
	return troncerrors.Invalid(err.Error())
}
