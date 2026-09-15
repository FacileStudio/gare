package server

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// DefaultPort is the default port the server listens on.
const DefaultPort = "8080"

// Config contains configuration options for Server.
type Config struct {
	Port          string
	Secret        string
	DeployHandler func(ctx context.Context, appName string) error
}

// Server handles incoming HTTP webhooks and dispatches deployments.
type Server struct {
	Port          string
	Secret        string
	DeployHandler func(ctx context.Context, appName string) error

	httpServer *http.Server
	listener   net.Listener
	locks      sync.Map
	mu         sync.Mutex
	wg         sync.WaitGroup
}

// New creates a new Server with the given configuration.
func New(cfg Config) *Server {
	return &Server{
		Port:          cfg.Port,
		Secret:        cfg.Secret,
		DeployHandler: cfg.DeployHandler,
	}
}

// NewServer creates a new Server with the given port, secret, and deploy handler.
func NewServer(port, secret string, handler func(ctx context.Context, appName string) error) *Server {
	return &Server{
		Port:          port,
		Secret:        secret,
		DeployHandler: handler,
	}
}

func (s *Server) resolveAddr(addr string) string {
	if addr != "" {
		if !strings.Contains(addr, ":") {
			return ":" + addr
		}
		return addr
	}
	if s.Port == "" {
		return ":" + DefaultPort
	}
	if strings.Contains(s.Port, ":") {
		return s.Port
	}
	return ":" + s.Port
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

// Shutdown gracefully stops the HTTP server.
func (s *Server) Shutdown(ctx context.Context) error {
	s.mu.Lock()
	srv := s.httpServer
	s.mu.Unlock()
	if srv == nil {
		return nil
	}
	if err := srv.Shutdown(ctx); err != nil {
		return err
	}
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
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
