package apps

import (
	"net/http"

	"github.com/FacileStudio/gare/apps/api/internal/apikit"
	"github.com/FacileStudio/tronc/httpjson"
	"github.com/go-chi/chi/v5"
)

// RegisterRoutes mounts the app routes on the /api router, every one behind requireAuth.
func RegisterRoutes(router chi.Router, service *Service, requireAuth func(http.Handler) http.Handler) {
	router.Group(func(r chi.Router) {
		r.Use(requireAuth)
		registerReads(r, service)
		registerChanges(r, service)
	})
}

// registerReads mounts the routes that only read.
func registerReads(r chi.Router, service *Service) {
	r.Get("/apps", service.handleList)
	r.Get("/apps/{name}", service.handleGet)
	r.Get("/apps/{name}/logs", service.handleLogs)
}

// registerChanges mounts the routes that change an app.
func registerChanges(r chi.Router, service *Service) {
	r.Post("/apps", service.handleCreate)
	r.Delete("/apps/{name}", service.handleDestroy)
	r.Post("/apps/{name}/start", service.lifecycleHandler(actionStart))
	r.Post("/apps/{name}/stop", service.lifecycleHandler(actionStop))
	r.Post("/apps/{name}/restart", service.lifecycleHandler(actionRestart))
	r.Put("/apps/{name}/tags", service.handleTags)
}

// handleList answers every app.
func (s *Service) handleList(w http.ResponseWriter, r *http.Request) {
	list, err := s.list(r.Context())
	apikit.Respond(w, http.StatusOK, list, err)
}

// handleGet answers one app.
func (s *Service) handleGet(w http.ResponseWriter, r *http.Request) {
	app, err := s.view(r.Context(), apikit.Name(r))
	apikit.Respond(w, http.StatusOK, app, err)
}

// handleLogs streams the app's journal. An unknown app or a bad count is answered as JSON before
// the stream opens; after that the stream runs until the client goes away.
func (s *Service) handleLogs(w http.ResponseWriter, r *http.Request) {
	name := apikit.Name(r)
	if _, err := s.deps.Known(name); err != nil {
		httpjson.WriteError(w, err)
		return
	}
	lines, err := logLines(r.URL.Query().Get("lines"))
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}
	stream, err := apikit.OpenStream(w)
	if err != nil {
		return
	}
	s.streamLogs(r.Context(), name, lines, stream)
}
