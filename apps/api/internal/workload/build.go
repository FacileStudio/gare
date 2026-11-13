package workload

import (
	"context"
	"fmt"

	"github.com/FacileStudio/gare/apps/api/internal/compose"
	"github.com/FacileStudio/gare/apps/api/internal/podman"
)

// runBuildCommand runs the repository's configured build command, if it has one, streaming its
// output to the app's writers.
func runBuildCommand(ctx context.Context, app App) error {
	command := app.Config.BuildCmd
	if command == "" {
		return nil
	}
	app.Report.Step(fmt.Sprintf("Running build command: %s", command))
	if err := podman.RunBuildCommand(ctx, app.RepoDir(), command, app.Stdout, app.Stderr); err != nil {
		return fmt.Errorf("build command failed: %w", err)
	}
	return nil
}

// warnStrayComposeFile tells the operator a repository ships a compose file that a workload of
// another type ignores, because gare never guesses the type from what the repository contains.
func warnStrayComposeFile(app App) {
	if detected := compose.DetectFile(app.RepoDir()); detected != "" {
		app.Report.Warn(fmt.Sprintf("compose file %s found but the workload type is not compose; "+
			"set type: compose in gare.yml to deploy it", detected))
	}
}

// bestEffort runs one teardown step and reports its failure as a warning, because a destroy keeps
// going through whatever is already gone rather than abandoning the rest of the teardown.
func bestEffort(app App, label string, action func() error) {
	if err := action(); err != nil {
		app.Report.Warn(fmt.Sprintf("%s returned error: %v", label, err))
	}
}
