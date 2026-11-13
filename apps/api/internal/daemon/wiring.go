package daemon

import (
	"io"
	"log/slog"

	"github.com/FacileStudio/gare/apps/api/internal/apikit"
	"github.com/FacileStudio/gare/apps/api/internal/engine"
	"github.com/FacileStudio/gare/apps/api/internal/server"
	"github.com/FacileStudio/gare/apps/api/internal/storage"
	"github.com/FacileStudio/gare/apps/api/modules/apps"
	"github.com/FacileStudio/gare/apps/api/modules/auth"
	"github.com/FacileStudio/gare/apps/api/modules/deploys"
	"github.com/FacileStudio/gare/apps/api/modules/domains"
	"github.com/FacileStudio/gare/apps/api/modules/env"
	"github.com/go-chi/chi/v5"
)

// logSink reports the engine's progress to the daemon's log. Subprocess output is dropped here:
// each deploy's own log keeps it, which is where the API and the CLI read it.
type logSink struct {
	logger *slog.Logger
}

// Event logs one progress line with its app.
func (s logSink) Event(app string, _ engine.Kind, msg string) {
	s.logger.Info(msg, slog.String("app", app))
}

// Output discards subprocess output.
func (logSink) Output(string) (io.Writer, io.Writer) { return io.Discard, io.Discard }

// apiDeps builds the engine every module drives and makes the webhook intake deploy through it
// too, so a pushed deploy and an API deploy share one queue and one history.
func apiDeps(cfg Config, intake *server.Server, logger *slog.Logger) apikit.Deps {
	store := cfg.Store
	if store == nil {
		store = storage.Default()
	}
	eng := engine.New(store, logSink{logger: logger}, cfg.Settings)
	intake.UseDeployer(eng)
	return apikit.Deps{Engine: eng, Store: store, Queue: intake, AllowLocalRepos: cfg.allowLocalRepos}
}

// mountAPI registers every /api module: porte's sign-in first, then the operator routes behind the
// allowlisted session.
func mountAPI(service *auth.Service, deps apikit.Deps) func(chi.Router) error {
	return func(api chi.Router) error {
		if err := auth.RegisterRoutes(api, service); err != nil {
			return err
		}
		requireAuth := service.RequireAuth()
		apps.RegisterRoutes(api, apps.NewService(deps), requireAuth)
		deploys.RegisterRoutes(api, deploys.NewService(deps), requireAuth)
		env.RegisterRoutes(api, env.NewService(deps), requireAuth)
		domains.RegisterRoutes(api, domains.NewService(deps), requireAuth)
		return nil
	}
}
