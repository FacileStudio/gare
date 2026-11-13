package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/FacileStudio/gare/apps/api/internal/appname"
)

type destroyOptions struct {
	yes     bool
	volumes bool
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
	if err := newEngine(ctx).Destroy(ctx, name, opts.volumes); err != nil {
		return err
	}
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
