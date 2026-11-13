package env

import (
	"net/http"

	"github.com/FacileStudio/gare/apps/api/internal/apikit"
	"github.com/FacileStudio/tronc/httpjson"
	"github.com/go-chi/chi/v5"
)

// RegisterRoutes mounts the env routes on the /api router, every one behind requireAuth.
func RegisterRoutes(router chi.Router, service *Service, requireAuth func(http.Handler) http.Handler) {
	router.With(requireAuth).Get("/apps/{name}/env", service.handleGet)
	router.With(requireAuth).Patch("/apps/{name}/env", service.handlePatch)
}

// handleGet answers the masked list, or one variable in clear when ?reveal names it.
func (s *Service) handleGet(w http.ResponseWriter, r *http.Request) {
	name := apikit.Name(r)
	if key := r.URL.Query().Get("reveal"); key != "" {
		revealed, err := s.reveal(name, key)
		apikit.Respond(w, http.StatusOK, revealed, err)
		return
	}
	list, err := s.list(name)
	apikit.Respond(w, http.StatusOK, list, err)
}

// handlePatch merges a change and answers the variables. The restart it may cause runs detached
// from the request, so a client that gives up does not leave the app stopped.
func (s *Service) handlePatch(w http.ResponseWriter, r *http.Request) {
	var change EnvPatch
	if err := httpjson.DecodeJSON(w, r, &change); err != nil {
		httpjson.WriteError(w, err)
		return
	}
	ctx, cancel := apikit.Detach(r)
	defer cancel()
	list, err := s.patch(ctx, apikit.Name(r), change)
	apikit.Respond(w, http.StatusOK, list, err)
}
