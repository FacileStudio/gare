package cli

import (
	"context"
	"errors"
	"io/fs"

	"github.com/FacileStudio/gare/apps/api/internal/appname"
	"github.com/FacileStudio/gare/apps/api/internal/deploys"
	"github.com/FacileStudio/gare/apps/api/internal/engine"
	"github.com/FacileStudio/gare/apps/api/internal/server"
	"github.com/FacileStudio/gare/apps/api/internal/storage"
)

// storedBranch reports the branch an app deploys from its stored config, so the webhook server
// can skip pushes to other branches. A missing config means gare does not manage the app.
func storedBranch(app string) (string, error) {
	cfg, err := storage.Default().Load(app)
	if errors.Is(err, fs.ErrNotExist) {
		return "", server.ErrUnknownApp
	}
	if err != nil {
		return "", err
	}
	return cfg.Branch, nil
}

// newWebhookServer assembles the webhook server from the operator's settings, failing before
// anything listens when the secret or the concurrency cap is missing or malformed.
func newWebhookServer(opts serverOptions) (*server.Server, error) {
	secret, err := resolveWebhookSecret(opts.secretFile)
	if err != nil {
		return nil, err
	}
	maxParallel, err := resolveMaxParallel()
	if err != nil {
		return nil, err
	}
	return server.New(server.Config{
		Secret:        secret,
		BranchFor:     storedBranch,
		MaxParallel:   maxParallel,
		DeployHandler: webhookDeploy(opts.settings, RunDeploy),
	}), nil
}

// deployFunc runs one deploy of an app for a trigger; RunDeploy is the real one.
type deployFunc func(ctx context.Context, name string, trigger deploys.Trigger) error

// webhookDeploy returns the handler the webhook server calls for each pushed app. The server hands
// it a context of its own, which carries none of the command's settings, so the handler attaches the
// settings the daemon resolved at start: a pushed deploy then authenticates to git and logs exactly
// as a typed `gare deploy` would.
func webhookDeploy(settings engine.Settings, deploy deployFunc) func(context.Context, string) error {
	return func(ctx context.Context, appName string) error {
		if err := appname.Validate(appName); err != nil {
			return err
		}
		return deploy(withSettings(ctx, settings), appName, deploys.TriggerWebhook)
	}
}
