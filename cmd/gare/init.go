package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"github.com/FacileStudio/gare/internal/caddy"
	"github.com/FacileStudio/gare/internal/storage"
	"github.com/FacileStudio/gare/internal/systemd"
	"github.com/spf13/cobra"
)

// NewInitCmd builds the init command.
func NewInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Validate prerequisites and initialize base directories",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()
			return runInit(ctx)
		},
	}
}

func runInit(ctx context.Context) error {
	if err := checkPrereqs(); err != nil {
		return err
	}
	checkPodmanWorkloads(ctx)
	checkLingerStatus(ctx)
	checkPauseSetup()
	checkComposeProvider(ctx)
	if err := initDirectories(); err != nil {
		return err
	}
	checkCaddyPermissions()
	return nil
}

func checkPrereqs() error {
	prereqs := []string{"podman", "caddy", "git"}
	for _, tool := range prereqs {
		if _, err := exec.LookPath(tool); err != nil {
			printError(fmt.Sprintf("Required prerequisite %q is not in PATH", tool))
			return fmt.Errorf("prerequisite %q not found", tool)
		}
		printSuccess("Found " + tool)
	}
	return nil
}

func checkLingerStatus(ctx context.Context) {
	lingering, err := systemd.CheckLinger(ctx, "")
	if err != nil || !lingering {
		printWarning("Lingering is not enabled for the current user")
		fmt.Printf("\nRun to enable lingering:\n  loginctl enable-linger $USER\n\n")
		return
	}
	printSuccess("User linger is enabled")
}

func checkPauseSetup() {
	if _, err := exec.LookPath("catatonit"); err == nil {
		printSuccess("Found catatonit init binary")
		return
	}
	home, err := os.UserHomeDir()
	if err != nil {
		printWarning("Could not determine home directory to check containers.conf")
		return
	}
	confPath := filepath.Join(home, ".config", "containers", "containers.conf")
	data, err := os.ReadFile(confPath)
	if err == nil && strings.Contains(string(data), "infra_image") {
		printSuccess("Found infra_image configured in " + confPath)
		return
	}
	printWarning("catatonit not found in PATH")
	fmt.Printf("\nTo configure pod pause init, install catatonit or add to %s:\n  [engine]\n  infra_image = \"registry.k8s.io/pause:3.9\"\n\n", confPath)
}

func initDirectories() error {
	baseDir := storage.DefaultBaseDir()
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return fmt.Errorf("failed to create storage directory %s: %w", baseDir, err)
	}
	printSuccess("Created storage directory: " + baseDir)

	userUnitDir := systemd.DefaultUserUnitDir()
	if err := os.MkdirAll(userUnitDir, 0755); err != nil {
		return fmt.Errorf("failed to create systemd directory %s: %w", userUnitDir, err)
	}
	printSuccess("Created systemd directory: " + userUnitDir)
	return nil
}

func checkCaddyPermissions() {
	confDir := caddy.ResolveConfDir()
	testFile := filepath.Join(confDir, ".gare_test")
	f, err := os.Create(testFile)
	if err != nil {
		if !attemptSudoCaddySetup(confDir, filepath.Dir(confDir)) {
			return
		}
	} else {
		f.Close()
		if rmErr := os.Remove(testFile); rmErr != nil {
			printWarning(fmt.Sprintf("Could not remove test file: %v", rmErr))
		}
	}
	if err := caddy.EnsureCaddyfile(); err != nil {
		printWarning(fmt.Sprintf("Could not configure Caddyfile: %v", err))
		return
	}
	caddyfilePath := caddy.ResolveCaddyfilePath()
	data, err := os.ReadFile(caddyfilePath)
	if err != nil || !strings.Contains(string(data), "conf.d") {
		printWarning("Caddyfile missing conf.d import directive")
		return
	}
	printSuccess("Verified snippet directory: " + confDir)
	printSuccess("Verified Caddyfile import directive: " + caddyfilePath)
	printSuccess("Gare initialized successfully")
}

func attemptSudoCaddySetup(confDir, parentDir string) bool {
	printInfo("Requesting sudo to configure " + confDir + " permissions...")
	sudoPath, err := exec.LookPath("sudo")
	if err != nil {
		printWarning(fmt.Sprintf("Directory %s is not writable (sudo not found)", confDir))
		fmt.Printf("\nTo configure Caddy permissions manually, run:\n  sudo mkdir -p %s && sudo chown -R $USER: %s\n\n", confDir, parentDir)
		return false
	}
	u, err := user.Current()
	if err != nil {
		return false
	}
	script := fmt.Sprintf("mkdir -p %s && chown -R %s: %s", confDir, u.Username, parentDir)
	cmd := exec.Command(sudoPath, "sh", "-c", script)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		printWarning(fmt.Sprintf("Directory %s is not writable", confDir))
		fmt.Printf("\nTo configure Caddy permissions manually, run:\n  sudo mkdir -p %s && sudo chown -R $USER: %s\n\n", confDir, parentDir)
		return false
	}
	return true
}
