package server

import (
	"log/slog"
	"net/http"

	"github.com/FacileStudio/tronc/apiref"
	troncerrors "github.com/FacileStudio/tronc/errors"
	"github.com/FacileStudio/tronc/httpjson"
	"github.com/FacileStudio/tronc/httpx"
	"github.com/FacileStudio/tronc/middleware"
	"github.com/FacileStudio/tronc/spa"
	"github.com/go-chi/chi/v5"
)

// Routes is what the daemon mounts beside the webhook intake. API registers
// everything under /api, Docs feeds the reference at /docs, and ClientDir is
// the built web client, served last and only when it holds an index.html.
type Routes struct {
	API       func(chi.Router) error
	Docs      apiref.Config
	ClientDir string
	Logger    *slog.Logger
}

// Router assembles the daemon's HTTP surface in the suite's shape. /health
// and the webhook paths stay at the root and go to the same handler the
// webhook-only daemon serves, so forges keep their URLs and their answers.
func (s *Server) Router(routes Routes) (chi.Router, error) {
	router := httpx.NewRouter(httpx.Config{Logger: routes.Logger, Security: security(routes.ClientDir, routes.Logger)})
	intake := s.Handler()
	router.Handle("/health", intake)
	router.Handle("/webhook", intake)
	router.Handle("/webhook/*", intake)
	apiref.Mount(router, routes.Docs)

	var apiErr error
	router.Route("/api", func(api chi.Router) {
		api.NotFound(func(w http.ResponseWriter, _ *http.Request) {
			httpjson.WriteError(w, troncerrors.NotFound("not found"))
		})
		apiErr = routes.API(api)
	})
	if spa.Available(routes.ClientDir) {
		router.NotFound(spa.Handler(spa.Config{Dir: routes.ClientDir}).ServeHTTP)
	}
	return router, apiErr
}

// security allows the client's inline scripts by hash. The policy is report
// only in tronc, so a hash that cannot be computed is logged, not fatal.
func security(clientDir string, logger *slog.Logger) middleware.SecurityConfig {
	if !spa.Available(clientDir) {
		return middleware.SecurityConfig{}
	}
	hashes, err := spa.InlineScriptHashes(clientDir)
	if err != nil && logger != nil {
		logger.Warn("inline script hashes unavailable, the CSP will report the client's scripts", slog.Any("error", err))
	}
	return middleware.SecurityConfig{CSPSources: map[string][]string{"script-src": hashes}}
}
