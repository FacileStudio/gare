package engine

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/FacileStudio/gare/apps/api/internal/appconfig"
	"github.com/FacileStudio/gare/apps/api/internal/git"
	"github.com/FacileStudio/gare/apps/api/internal/storage"
	"github.com/FacileStudio/gare/apps/api/internal/systemd"
	"github.com/FacileStudio/gare/apps/api/internal/workload"
)

// CreateRequest is what a new app is created from: its name, which is derived from the repository
// when empty, the repository and branch to clone, and the settings the operator passed, which
// outrank the repository's gare.yml.
type CreateRequest struct {
	Name   string
	Repo   string
	Branch string
	Flags  appconfig.Flags
}

// Create clones a new app's repository, settles its configuration and writes its workload
// artifacts, holding the app's lock throughout. A failed clone leaves no storage behind.
func (e *Engine) Create(ctx context.Context, req CreateRequest) (*storage.AppConfig, error) {
	if req.Name == "" {
		req.Name = DeriveName(req.Repo)
	}
	if err := validateCreate(req); err != nil {
		return nil, err
	}
	unlock, err := e.waitForLock(ctx, req.Name)
	if err != nil {
		return nil, err
	}
	defer unlock()
	appDir := e.store.AppDir(req.Name)
	if _, err := os.Stat(storage.GetConfigPath(appDir)); err == nil {
		return nil, fmt.Errorf("app %q already exists at %s", req.Name, appDir)
	}
	if err := e.clone(ctx, appDir, req); err != nil {
		return nil, errors.Join(err, e.store.Delete(req.Name))
	}
	return e.setup(ctx, appDir, req)
}

// DeriveName returns the app name a repository URL implies: its last path segment, lowercased,
// without a trailing slash or .git suffix.
func DeriveName(repoURL string) string {
	cleaned := strings.TrimSpace(repoURL)
	cleaned = strings.TrimSuffix(cleaned, "/")
	cleaned = strings.TrimSuffix(cleaned, ".git")
	if idx := strings.LastIndexAny(cleaned, "/:"); idx != -1 {
		cleaned = cleaned[idx+1:]
	}
	return strings.ToLower(cleaned)
}

// clone clones the app's repository into its storage directory.
func (e *Engine) clone(ctx context.Context, appDir string, req CreateRequest) error {
	repoDir := storage.GetRepoDir(appDir)
	e.report(req.Name).Step(fmt.Sprintf("Cloning %s (%s) into %s...", req.Repo, req.Branch, repoDir))
	stdout, stderr := e.sink.Output(req.Name)
	return git.Clone(ctx, git.CloneOptions{
		RepoURL:   req.Repo,
		Branch:    req.Branch,
		TargetDir: repoDir,
		Auth:      e.settings.Auth,
		Stdout:    stdout,
		Stderr:    stderr,
	})
}

// setup settles a cloned app's configuration, allocates its port, writes its workload artifacts and
// records it.
func (e *Engine) setup(ctx context.Context, appDir string, req CreateRequest) (*storage.AppConfig, error) {
	cfg, w, err := e.resolveCreate(appDir, req)
	if err != nil {
		return nil, err
	}
	if cfg.Port, err = e.store.AvailablePort(cfg.Port); err != nil {
		return nil, fmt.Errorf("failed to discover port: %w", err)
	}
	if err := w.Prepare(ctx, e.workloadApp(cfg), workload.Create); err != nil {
		return nil, err
	}
	if err := e.saveCreated(cfg); err != nil {
		return nil, err
	}
	if err := systemd.DaemonReload(ctx); err != nil {
		e.report(cfg.Name).Warn(fmt.Sprintf("daemon-reload error: %v", err))
	}
	return cfg, nil
}

// resolveCreate decides a new app's configuration from the flags and the cloned repository's
// gare.yml, with the precedence appconfig.Resolve applies on every deploy, then lets its workload
// type settle what only the checkout can tell.
func (e *Engine) resolveCreate(appDir string, req CreateRequest) (*storage.AppConfig, workload.Workload, error) {
	gf, err := storage.LoadGareFile(storage.GetRepoDir(appDir))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load gare configuration: %w", err)
	}
	cfg, err := appconfig.Resolve(req.Flags, gf, nil)
	if err != nil {
		return nil, nil, err
	}
	w, err := workload.For(cfg)
	if err != nil {
		return nil, nil, err
	}
	cfg.Name, cfg.RepoURL, cfg.Branch = req.Name, req.Repo, req.Branch
	app := e.workloadApp(cfg)
	app.Dir = appDir
	return cfg, w, w.Settle(app, workload.Create)
}

// saveCreated records a newly created app, stamped with its creation time.
func (e *Engine) saveCreated(cfg *storage.AppConfig) error {
	cfg.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	if err := e.store.Save(cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}
	e.report(cfg.Name).Done(fmt.Sprintf("App %q successfully created on port %d", cfg.Name, cfg.Port))
	return nil
}
