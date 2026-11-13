package main

import (
	"os"

	"github.com/FacileStudio/gare/apps/api/cli"
)

// main exits with the status run reports, so deferred work in run finishes before the process ends.
func main() {
	os.Exit(run())
}

// run executes the gare command tree and maps its outcome to a process exit status.
func run() int {
	if err := cli.Execute(version); err != nil {
		return 1
	}
	return 0
}
