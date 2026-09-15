package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/FacileStudio/gare/internal/server"
	"github.com/FacileStudio/gare/internal/storage"
	"github.com/spf13/cobra"
)

type serverOptions struct {
	port   string
	secret string
}

// NewServerCmd builds the server command.
func NewServerCmd() *cobra.Command {
	opts := serverOptions{}
	cmd := &cobra.Command{
		Use:   "server --port <port> --secret <token>",
		Short: "Run the webhook HTTP daemon",
		RunE: func(c *cobra.Command, args []string) error {
			return runWebhookDaemon(opts)
		},
	}

	cmd.Flags().StringVarP(&opts.port, "port", "p", "8080", "Port to listen on")
	cmd.Flags().StringVarP(&opts.secret, "secret", "s", "", "Webhook secret token")
	return cmd
}

func runWebhookDaemon(opts serverOptions) error {
	if opts.secret == "" {
		printWarning("No --secret provided. Webhooks will not be authenticated.")
	}

	srv := server.New(server.Config{
		Port:   opts.port,
		Secret: opts.secret,
		DeployHandler: func(ctx context.Context, appName string) error {
			if err := storage.ValidateAppName(appName); err != nil {
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
