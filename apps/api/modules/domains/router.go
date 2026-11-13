package domains

import (
	"net/http"

	"github.com/FacileStudio/gare/apps/api/internal/apikit"
	"github.com/FacileStudio/tronc/httpjson"
	"github.com/go-chi/chi/v5"
)

// RegisterRoutes mounts the domain routes on the /api router, every one behind requireAuth.
func RegisterRoutes(router chi.Router, service *Service, requireAuth func(http.Handler) http.Handler) {
	router.Group(func(r chi.Router) {
		r.Use(requireAuth)
		r.Get("/apps/{name}/domains", service.handleList)
		r.Post("/apps/{name}/domains", service.handleAdd)
		r.Delete("/apps/{name}/domains/{hostname}", service.handleRemove)
	})
}

// handleList answers the app's hostnames.
func (s *Service) handleList(w http.ResponseWriter, r *http.Request) {
	list, err := s.list(apikit.Name(r))
	apikit.Respond(w, http.StatusOK, list, err)
}

// handleAdd adds a hostname. The ingress reload it may cause runs detached from the request.
func (s *Service) handleAdd(w http.ResponseWriter, r *http.Request) {
	var body AddRequest
	if err := httpjson.DecodeJSON(w, r, &body); err != nil {
		httpjson.WriteError(w, err)
		return
	}
	ctx, cancel := apikit.Detach(r)
	defer cancel()
	list, err := s.add(ctx, apikit.Name(r), body.Hostname)
	apikit.Respond(w, http.StatusOK, list, err)
}

// handleRemove removes a hostname.
func (s *Service) handleRemove(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := apikit.Detach(r)
	defer cancel()
	list, err := s.remove(ctx, apikit.Name(r), chi.URLParam(r, "hostname"))
	apikit.Respond(w, http.StatusOK, list, err)
}
