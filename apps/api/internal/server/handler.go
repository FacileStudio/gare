package server

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/FacileStudio/gare/apps/api/internal/appname"
)

// Handler returns the HTTP handler with all registered server routes.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("POST /webhook", s.handleWebhook)
	mux.HandleFunc("POST /webhook/{app}", s.handleWebhook)
	return mux
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (s *Server) handleWebhook(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20)
	defer r.Body.Close()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	if !s.verifyAuth(r, body) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	appName := s.extractAppName(r, body)
	if appName == "" {
		http.Error(w, "Invalid or missing app name", http.StatusBadRequest)
		return
	}

	s.answerDelivery(w, r, appName, body)
}

// answerDelivery decides what a verified delivery for appName does: ignore an event that is not a
// push, refuse an unknown app, skip a push to another branch, or queue a deploy and say how.
func (s *Server) answerDelivery(w http.ResponseWriter, r *http.Request, appName string, body []byte) {
	d, err := readDelivery(r, body)
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	if d.ignore != "" {
		writeAnswer(w, http.StatusOK, "ignored: "+d.ignore)
		return
	}
	if status, answer := s.matchBranch(appName, d); status != 0 {
		writeAnswer(w, status, answer)
		return
	}
	answer, ok := s.queue.enqueue(appName)
	if !ok {
		http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
		return
	}
	writeAnswer(w, http.StatusAccepted, answer)
}

// writeAnswer sends a plain-text answer, which is what a forge shows in its delivery log.
func writeAnswer(w http.ResponseWriter, status int, answer string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	io.WriteString(w, answer)
}

func (s *Server) extractAppName(r *http.Request, body []byte) string {
	app := r.PathValue("app")
	if app == "" {
		trimmed := strings.Trim(strings.TrimPrefix(r.URL.Path, "/webhook"), "/")
		if trimmed != "" {
			app = trimmed
		}
	}
	if app == "" && len(body) > 0 {
		var payload struct {
			App string `json:"app"`
		}
		if err := json.Unmarshal(body, &payload); err == nil {
			app = payload.App
		}
	}
	if appname.Validate(app) != nil {
		return ""
	}
	return app
}
