package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/FacileStudio/gare/internal/appname"
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

	if s.secret != "" && !s.verifyAuth(r, body) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	appName := s.extractAppName(r, body)
	if appName == "" {
		http.Error(w, "Invalid or missing app name", http.StatusBadRequest)
		return
	}

	s.dispatchDeploy(appName)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Deploy queued for %s", appName)
}

func (s *Server) dispatchDeploy(appName string) {
	lock := s.getAppLock(appName)
	s.wg.Go(func() {
		lock.Lock()
		defer lock.Unlock()
		if s.deployHandler != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
			defer cancel()
			if err := s.deployHandler(ctx, appName); err != nil {
				fmt.Fprintf(os.Stderr, "deploy error for %s: %v\n", appName, err)
			}
		}
	})
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

func (s *Server) getAppLock(appName string) *sync.Mutex {
	val, _ := s.locks.LoadOrStore(appName, &sync.Mutex{})
	return val.(*sync.Mutex)
}
