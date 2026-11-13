package server

import (
	"encoding/json"
	"errors"
	"net/http"
)

// ErrUnknownApp is what Config.BranchFor returns for an app gare does not manage, so the handler
// can answer 404 instead of queueing a deploy that is bound to fail.
var ErrUnknownApp = errors.New("unknown app")

// Event header values that mean "code was pushed". Every other event a forge sends, such as
// GitHub's ping or a GitLab tag push, is acknowledged and ignored.
const (
	githubPushEvent = "push"
	gitlabPushEvent = "Push Hook"
)

// delivery is what a webhook asks for: a push carrying a ref, an event to ignore, or neither,
// which is a hand-made trigger that deploys unconditionally.
type delivery struct {
	push   bool
	ref    string
	ignore string
}

// readDelivery classifies a request by its forge event header. A request with no event header is
// a manual trigger, which keeps `curl` with a signature working as before.
func readDelivery(r *http.Request, body []byte) (delivery, error) {
	github := r.Header.Get("X-GitHub-Event")
	gitlab := r.Header.Get("X-Gitlab-Event")
	switch {
	case github == githubPushEvent, gitlab == gitlabPushEvent:
		ref, err := pushRef(body)
		return delivery{push: true, ref: ref}, err
	case github != "":
		return delivery{ignore: github}, nil
	case gitlab != "":
		return delivery{ignore: gitlab}, nil
	}
	return delivery{}, nil
}

// pushRef reads the ref a push event names. Both GitHub and GitLab put it at the top level as
// `refs/heads/<branch>`; a push without one cannot be matched to a branch, so it is refused.
func pushRef(body []byte) (string, error) {
	var payload struct {
		Ref string `json:"ref"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", err
	}
	if payload.Ref == "" {
		return "", errors.New("push event without a ref")
	}
	return payload.Ref, nil
}

// matchBranch answers whether app's deploy should run for d. It returns a non-zero status and a
// body when the delivery stops here: 404 for an app gare does not manage, 200 for a push to a
// branch the app does not deploy. An empty stored branch accepts any ref.
func (s *Server) matchBranch(app string, d delivery) (int, string) {
	if s.branchFor == nil {
		return 0, ""
	}
	branch, err := s.branchFor(app)
	if errors.Is(err, ErrUnknownApp) {
		return http.StatusNotFound, "unknown app"
	}
	if err != nil {
		return http.StatusInternalServerError, "Internal Server Error"
	}
	if !d.push || branch == "" || d.ref == "refs/heads/"+branch {
		return 0, ""
	}
	return http.StatusOK, "ignored: " + d.ref
}
