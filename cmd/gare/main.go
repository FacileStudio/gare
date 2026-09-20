package main

import (
	"os"
)

var version = "0.11.0"

func main() {
	if err := Execute(version); err != nil {
		os.Exit(1)
	}
}
