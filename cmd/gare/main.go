package main

import (
	"os"
)

var version = "0.13.3"

func main() {
	if err := Execute(version); err != nil {
		os.Exit(1)
	}
}
