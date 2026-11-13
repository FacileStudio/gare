package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/FacileStudio/gare/internal/appname"
	"github.com/FacileStudio/gare/internal/storage"
)

type destroyOptions struct {
	yes     bool
	volumes bool
}

// destroyPlan names everything one destroy acts on, so the teardown steps agree on the app and on
// whether its data volumes go with it.
type destroyPlan struct {
	name     string
	appDir   string
	workload storage.WorkloadType
	cfg      *storage.AppConfig
	volumes  bool
}

// runDestroy tears an application down once the operator has confirmed it. A destroy cannot be
// undone, so it never runs on a typo: it needs --yes, or the app name typed back on a terminal.
func runDestroy(ctx context.Context, name string, opts destroyOptions) error {
	if err := appname.Validate(name); err != nil {
		return err
	}
	if err := confirmDestroy(name, opts.yes, os.Stdin, os.Stdout, stdinIsTerminal()); err != nil {
		return err
	}
	appDir := storage.GetAppDir(storage.DefaultBaseDir(), name)
	cfg, workload, err := resolveDestroyWorkload(appDir)
	if err != nil {
		printWarning(fmt.Sprintf("%v — tearing down only what does not depend on the workload type, so a workload may keep running and its image survives: check them with `podman ps` and `podman images`", err))
	}
	teardownServices(ctx, name)
	removeArtifacts(ctx, destroyPlan{name: name, appDir: appDir, workload: workload, cfg: cfg, volumes: opts.volumes})
	reloadDaemons(ctx)
	printSuccess(fmt.Sprintf("App %s completely destroyed", name))
	return nil
}

// confirmDestroy passes when the operator said --yes or typed the app name back. Without a
// terminal to ask on, it refuses rather than guessing.
func confirmDestroy(name string, yes bool, in io.Reader, out io.Writer, interactive bool) error {
	if yes {
		return nil
	}
	if !interactive {
		return fmt.Errorf("refusing to destroy %q without confirmation: pass --yes", name)
	}
	fmt.Fprintf(out, "This removes %s with its unit, ingress and files. Type the app name to confirm: ", name)
	line, _ := bufio.NewReader(in).ReadString('\n')
	if strings.TrimSpace(line) != name {
		return fmt.Errorf("confirmation did not match %q; nothing was destroyed", name)
	}
	return nil
}

func stdinIsTerminal() bool {
	info, err := os.Stdin.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

// volumeNote says whether a compose teardown takes the stack's named volumes with it.
func volumeNote(volumes bool) string {
	if volumes {
		return " including its volumes"
	}
	return ", keeping its volumes"
}
