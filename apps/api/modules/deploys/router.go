package deploys

import (
	"net/http"

	"github.com/FacileStudio/gare/apps/api/internal/apikit"
	"github.com/FacileStudio/tronc/httpjson"
	"github.com/go-chi/chi/v5"
)

// RegisterRoutes mounts the deploy routes on the /api router, every one behind requireAuth.
func RegisterRoutes(router chi.Router, service *Service, requireAuth func(http.Handler) http.Handler) {
	router.Group(func(r chi.Router) {
		r.Use(requireAuth)
		registerReads(r, service)
		registerChanges(r, service)
	})
}

// registerReads mounts the history and the log stream.
func registerReads(r chi.Router, service *Service) {
	r.Get("/apps/{name}/deploys", service.handleList)
	r.Get("/apps/{name}/deploys/{id}", service.handleGet)
	r.Get("/apps/{name}/deploys/{id}/log", service.handleLog)
}

// registerChanges mounts deploy and rollback.
func registerChanges(r chi.Router, service *Service) {
	r.Post("/apps/{name}/deploy", service.handleDeploy)
	r.Post("/apps/{name}/rollback", service.handleRollback)
}

// handleList answers the app's deploys, newest first.
func (s *Service) handleList(w http.ResponseWriter, r *http.Request) {
	list, err := s.list(apikit.Name(r))
	apikit.Respond(w, http.StatusOK, list, err)
}

// handleGet answers one deploy.
func (s *Service) handleGet(w http.ResponseWriter, r *http.Request) {
	rec, err := s.get(apikit.Name(r), chi.URLParam(r, "id"))
	apikit.Respond(w, http.StatusOK, rec, err)
}

// handleLog streams one deploy's log. An unknown app or deploy is answered as JSON before the
// stream opens; after that the stream replays, follows and ends with done.
func (s *Service) handleLog(w http.ResponseWriter, r *http.Request) {
	name, id := apikit.Name(r), chi.URLParam(r, "id")
	history, err := s.history(name)
	if err == nil {
		_, err = record(history, name, id)
	}
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	stream, err := apikit.OpenStream(w)
	if err != nil {
		return
	}
	streamLog(r.Context(), history, id, lastEventID(r), stream)
}
