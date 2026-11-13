package cli

import (
	"context"
	"fmt"
	"io"
	"sort"

	"github.com/FacileStudio/gare/apps/api/internal/engine"
)

// reportEnvChange prints how an environment change ended: the app restarted with it, or kept
// running without it because its unit was not active.
func reportEnvChange(name string, result *engine.EnvResult, err error) error {
	if err != nil {
		return err
	}
	if result.Restarted {
		printSuccess(fmt.Sprintf("Restarted %s.service (active)", name))
		return nil
	}
	printInfo(fmt.Sprintf("Service %s is not active (%s); environment saved", name, result.State))
	return nil
}

// runEnvList prints an application's environment variables, as JSON or as KEY=value lines.
func runEnvList(ctx context.Context, w io.Writer, name string, opts envListOptions) error {
	envs, err := newEngine(ctx).Env(name)
	if err != nil {
		return err
	}
	if opts.jsonOutput {
		return writeJSON(w, envs)
	}
	return outputEnvPlain(w, envs)
}

// outputEnvPlain prints variables sorted by key, one KEY=value per line.
func outputEnvPlain(w io.Writer, envs map[string]string) error {
	if len(envs) == 0 {
		fmt.Fprintln(w, "No environment variables configured")
		return nil
	}
	keys := make([]string, 0, len(envs))
	for k := range envs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Fprintf(w, "%s=%s\n", k, envs[k])
	}
	return nil
}
