package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/FacileStudio/gare/internal/appname"
	"github.com/FacileStudio/gare/internal/server"
	"github.com/spf13/cobra"
)

// webhookSecretEnv names the environment variable that carries the webhook secret. The secret is
// never a command-line flag, because argv is readable by every user on the host.
const webhookSecretEnv = "GARE_WEBHOOK_SECRET"

type serverOptions struct {
	port       string
	secretFile string
}

// NewServerCmd builds the server command.
func NewServerCmd() *cobra.Command {
	opts := serverOptions{}
	cmd := &cobra.Command{
		Use:   "server --port <port> [--secret-file <path>]",
		Short: "Run the webhook HTTP daemon",
		Long: "Run the webhook HTTP daemon. It refuses to start without a webhook secret, read from\n" +
			"--secret-file or from the " + webhookSecretEnv + " environment variable.",
		RunE: func(c *cobra.Command, args []string) error {
			return runWebhookDaemon(opts)
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

func runWebhookDaemon(opts serverOptions) error {
	secret, err := resolveWebhookSecret(opts.secretFile)
	if err != nil {
		return err
	}

	srv := server.New(server.Config{
		Secret: secret,
		DeployHandler: func(ctx context.Context, appName string) error {
			if err := appname.Validate(appName); err != nil {
				return err
			}
			return RunDeploy(ctx, appName)
		},
	})

	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(stopCh)

	serverErr := make(chan error, 1)
	addr := fmt.Sprintf(":%s", opts.port)
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
