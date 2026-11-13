package cli

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/FacileStudio/gare/apps/api/internal/appname"
	"github.com/FacileStudio/gare/apps/api/internal/daemon"
	"github.com/FacileStudio/gare/apps/api/internal/server"
	"github.com/FacileStudio/gare/apps/api/internal/storage"
	"github.com/spf13/cobra"
)

// webhookSecretEnv names the environment variable that carries the webhook secret. The secret is
// never a command-line flag, because argv is readable by every user on the host.
const webhookSecretEnv = "GARE_WEBHOOK_SECRET"

// maxParallelEnv names the environment variable that caps concurrent webhook deploys across apps,
// so an operator can size it to what the host can build at once.
const maxParallelEnv = "GARE_MAX_PARALLEL_DEPLOYS"

type serverOptions struct {
	port       string
	portSet    bool
	secretFile string
}

// NewServerCmd builds the server command.
func NewServerCmd() *cobra.Command {
	opts := serverOptions{}
	cmd := &cobra.Command{
		Use:   "server --port <port> [--secret-file <path>]",
		Short: "Run the webhook HTTP daemon",
		Long: "Run the webhook HTTP daemon. It refuses to start without a webhook secret, read from\n" +
			"--secret-file or from the " + webhookSecretEnv + " environment variable.\n\n" +
			"With OIDC_ISSUER set it also serves the API and the web UI on GARE_LISTEN, signing\n" +
			"operators in through the suite's identity provider.\n\n" +
			"Both modes listen on 127.0.0.1. --port changes only the port; GARE_LISTEN sets the whole\n" +
			"address and wins over --port.",
		RunE: func(c *cobra.Command, args []string) error {
			opts.portSet = c.Flags().Changed("port")
			return runServer(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.port, "port", "p", "8080", "Port to listen on")
	cmd.Flags().StringVar(&opts.secretFile, "secret-file", "", "File holding the webhook secret (default: $"+webhookSecretEnv+")")
	return cmd
}

// resolveWebhookSecret reads the webhook secret from the file the operator named, or else from the
// environment, and fails when neither yields one: a daemon without a secret would refuse every
// webhook, so it should not start at all.
func resolveWebhookSecret(secretFile string) (string, error) {
	if secretFile != "" {
		data, err := os.ReadFile(secretFile)
		if err != nil {
			return "", fmt.Errorf("failed to read the webhook secret: %w", err)
		}
		if secret := strings.TrimSpace(string(data)); secret != "" {
			return secret, nil
		}
		return "", fmt.Errorf("webhook secret file %s is empty", secretFile)
	}
	if secret := strings.TrimSpace(os.Getenv(webhookSecretEnv)); secret != "" {
		return secret, nil
	}
	return "", fmt.Errorf("no webhook secret: set %s or pass --secret-file", webhookSecretEnv)
}

// resolveMaxParallel reads the deploy concurrency cap from the environment. An unset variable
// keeps the server's default; a value that is not a positive integer stops the daemon, because
// silently falling back would hide the operator's typo behind a different limit.
func resolveMaxParallel() (int, error) {
	raw := strings.TrimSpace(os.Getenv(maxParallelEnv))
	if raw == "" {
		return server.DefaultMaxParallel, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		return 0, fmt.Errorf("%s must be a positive integer, got %q", maxParallelEnv, raw)
	}
	return n, nil
}

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
		Secret:      secret,
		BranchFor:   storedBranch,
		MaxParallel: maxParallel,
		DeployHandler: func(ctx context.Context, appName string) error {
			if err := appname.Validate(appName); err != nil {
				return err
			}
			return RunDeploy(ctx, appName)
		},
	}), nil
}

// runServer serves the API and the web UI next to the webhooks when OIDC is configured, and the
// webhooks alone otherwise, so an install that predates the API keeps running unchanged.
func runServer(opts serverOptions) error {
	cfg, err := daemon.Load()
	if err != nil {
		return err
	}
	if !cfg.Enabled() {
		printWarning("API and web UI disabled: OIDC_ISSUER is not set, serving webhooks only")
		return runWebhookDaemon(opts)
	}
	srv, err := newWebhookServer(opts)
	if err != nil {
		return err
	}
	if opts.portSet {
		cfg.Listen = daemon.ListenFromEnv(opts.port)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return daemon.Serve(ctx, cfg, srv, slog.Default())
}

func runWebhookDaemon(opts serverOptions) error {
	srv, err := newWebhookServer(opts)
	if err != nil {
		return err
	}

	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(stopCh)

	serverErr := make(chan error, 1)
	addr := daemon.ListenFromEnv(opts.port)
	printInfo(fmt.Sprintf("Starting webhook daemon on %s...", addr))
	go func() {
		serverErr <- srv.Start(addr)
	}()

	select {
	case err := <-serverErr:
		return err
	case <-stopCh:
		return shutdownServer(srv, serverErr)
	}
}

func shutdownServer(srv *server.Server, serverErr <-chan error) error {
	printInfo("Shutting down webhook server...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		printWarning(fmt.Sprintf("Shutdown error: %v", err))
		return err
	}
	return <-serverErr
}
