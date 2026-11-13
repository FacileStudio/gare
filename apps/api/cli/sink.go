package cli

import (
	"context"
	"io"
	"os"

	"github.com/FacileStudio/gare/apps/api/internal/engine"
	"github.com/FacileStudio/gare/apps/api/internal/storage"
)

// terminalSink renders the engine's progress on the operator's terminal, with the icons and
// streams gare has always used: steps and finished steps on stdout, warnings on stderr, and
// subprocess output passed straight through.
type terminalSink struct{}

// Event prints one progress line in the style its kind calls for.
func (terminalSink) Event(_ string, kind engine.Kind, msg string) {
	switch kind {
	case engine.KindDone:
		printSuccess(msg)
	case engine.KindWarn:
		printWarning(msg)
	case engine.KindDetail:
		printDetail(msg)
	default:
		printInfo(msg)
	}
}

// Output returns the process's own stdout and stderr, so git and podman stream to the terminal.
func (terminalSink) Output(string) (io.Writer, io.Writer) {
	return os.Stdout, os.Stderr
}

// newEngine returns the engine a command runs, on the operator's store and with the settings the
// command resolved. The webhook daemon attaches the settings it resolved at start to every deploy.
func newEngine(ctx context.Context) *engine.Engine {
	return engine.New(storage.Default(), terminalSink{}, settingsFrom(ctx))
}
