package server

import (
	"context"
	"errors"
	"net"
	"net/http"
	"sync"
	"time"
)

// DefaultPort is the default port the server listens on.
const DefaultPort = "8080"

// Config contains configuration options for Server. BranchFor reports the branch an app deploys,
// or ErrUnknownApp; when nil, every push matches. MaxParallel caps deploys across apps and
// defaults to DefaultMaxParallel.
type Config struct {
	Secret        string
	DeployHandler func(ctx context.Context, appName string) error
	BranchFor     func(app string) (string, error)
	MaxParallel   int
}

// Server handles incoming HTTP webhooks and dispatches deployments.
type Server struct {
	secret    string
	branchFor func(app string) (string, error)
	queue     *deployQueue

	httpServer *http.Server
	listener   net.Listener
	mu         sync.Mutex
}

// New creates a new Server with the given configuration.
func New(cfg Config) *Server {
	return &Server{
		secret:    cfg.Secret,
		branchFor: cfg.BranchFor,
		queue:     newDeployQueue(handlerDeployer{run: cfg.DeployHandler}, cfg.MaxParallel),
	}
}

// resolveAddr turns the caller's listen address into one net.Listen accepts, defaulting to the
// default port when the caller passes none. The port lives in the address alone so there is a
// single place to read it from.
func (s *Server) resolveAddr(addr string) string {
	if addr == "" {
		return ":" + DefaultPort
	}
	return addr
}

// Start listens and serves HTTP requests on the specified address.
func (s *Server) Start(addr string) error {
	ln, err := net.Listen("tcp", s.resolveAddr(addr))
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.listener = ln
	s.httpServer = &http.Server{
		Addr:              ln.Addr().String(),
		Handler:           s.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	s.mu.Unlock()

	err = s.httpServer.Serve(ln)
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

// Shutdown gracefully stops the HTTP server, then waits for running deploys while dropping the
// pending ones, so a restart never cuts a build in half nor starts one on its way out.
func (s *Server) Shutdown(ctx context.Context) error {
	s.mu.Lock()
	srv := s.httpServer
	s.mu.Unlock()
	if srv != nil {
		if err := srv.Shutdown(ctx); err != nil {
			return err
		}
	}
	s.queue.close()
	done := make(chan struct{})
	go func() {
		s.queue.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Addr returns the network address the server is listening on.
func (s *Server) Addr() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.listener != nil {
		return s.listener.Addr().String()
	}
	if s.httpServer != nil {
		return s.httpServer.Addr
	}
	return ""
}
