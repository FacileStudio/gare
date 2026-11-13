package apps

import (
	"net/http"

	"github.com/FacileStudio/gare/apps/api/internal/apikit"
	"github.com/FacileStudio/tronc/httpjson"
)

// handleCreate creates an app and answers it with 201. The clone runs detached from the request,
// so a client that gives up does not leave half a checkout behind.
func (s *Service) handleCreate(w http.ResponseWriter, r *http.Request) {
	var body CreateRequest
	if err := httpjson.DecodeJSON(w, r, &body); err != nil {
		httpjson.WriteError(w, err)
		return
	}
	ctx, cancel := apikit.Detach(r)
	defer cancel()
	app, err := s.create(ctx, body)
	apikit.Respond(w, http.StatusCreated, app, err)
}

// handleDestroy destroys an app and answers 204.
func (s *Service) handleDestroy(w http.ResponseWriter, r *http.Request) {
	var body DestroyRequest
	if err := httpjson.DecodeJSON(w, r, &body); err != nil {
		httpjson.WriteError(w, err)
		return
	}
	ctx, cancel := apikit.Detach(r)
	defer cancel()
	if err := s.destroy(ctx, apikit.Name(r), body.Confirm, r.URL.Query().Get("volumes") == "true"); err != nil {
		httpjson.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// lifecycleHandler answers one of start, stop and restart with the app as it then stands.
func (s *Service) lifecycleHandler(action string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := apikit.Detach(r)
		defer cancel()
		app, err := s.lifecycle(ctx, apikit.Name(r), action)
		apikit.Respond(w, http.StatusOK, app, err)
	}
}

// handleTags replaces the app's tags and answers the app.
func (s *Service) handleTags(w http.ResponseWriter, r *http.Request) {
	var body TagsRequest
	if err := httpjson.DecodeJSON(w, r, &body); err != nil {
		httpjson.WriteError(w, err)
		return
	}
	app, err := s.setTags(r.Context(), apikit.Name(r), body.Tags)
	apikit.Respond(w, http.StatusOK, app, err)
}
