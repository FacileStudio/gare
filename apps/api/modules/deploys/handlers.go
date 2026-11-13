package deploys

import (
	"net/http"

	"github.com/FacileStudio/gare/apps/api/internal/apikit"
	"github.com/FacileStudio/tronc/httpjson"
)

// handleDeploy queues a deploy and answers 202 with its record; the deploy runs in the background
// on the daemon's context, so the request returns at once.
func (s *Service) handleDeploy(w http.ResponseWriter, r *http.Request) {
	rec, err := s.deploy(apikit.Name(r))
	apikit.Respond(w, http.StatusAccepted, rec, err)
}

// handleRollback starts a rollback and answers 202 with its record. An empty body goes back to the
// newest successful deploy of another commit, and so does no body at all.
func (s *Service) handleRollback(w http.ResponseWriter, r *http.Request) {
	var body RollbackRequest
	if r.ContentLength != 0 {
		if err := httpjson.DecodeJSON(w, r, &body); err != nil {
			httpjson.WriteError(w, err)
			return
		}
	}
	rec, err := s.rollback(r.Context(), apikit.Name(r), body.DeployID)
	apikit.Respond(w, http.StatusAccepted, rec, err)
}
