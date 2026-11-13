package auth

import (
	"net/http"
	"strings"

	"github.com/FacileStudio/porte/directory"
	"github.com/FacileStudio/porte/kit"
	troncerrors "github.com/FacileStudio/tronc/errors"
	"github.com/FacileStudio/tronc/httpjson"
	"github.com/go-chi/chi/v5"
)

// RegisterRoutes mounts the sign-in routes on the /api router: porte's own
// session and OIDC routes, the provider's signed deletion push, and GET /me.
func RegisterRoutes(router chi.Router, service *Service) error {
	if err := kit.Mount(router, kit.Config{Sessions: service.sessions, OIDC: service.oidc}); err != nil {
		return err
	}
	registerDeletions(router, service)
	registerMe(router, service)
	return nil
}

// registerDeletions accepts the provider's signed user.deleted push on porte's
// path, relative to the /api router it is given.
func registerDeletions(router chi.Router, service *Service) {
	router.Method(http.MethodPost, strings.TrimPrefix(directory.DeletionsPath, "/api"), service.deletions.Handler())
}

// registerMe serves the signed-in operator behind the allowlisted middleware.
func registerMe(router chi.Router, service *Service) {
	router.With(service.RequireAuth()).Get("/me", service.handleMe)
}

// handleMe answers the operator bare, the shape revision 2 of the API
// contract fixed for the client.
func (s *Service) handleMe(w http.ResponseWriter, r *http.Request) {
	identity, ok := kit.FromContext(r.Context())
	if !ok {
		httpjson.WriteError(w, troncerrors.Unauthorized("missing auth"))
		return
	}
	httpjson.WriteJSON(w, http.StatusOK, s.controller.me(identity))
}
