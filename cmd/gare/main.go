package main

import (
	"os"
)

var version = "0.8.5"

func main() {
	if err := Execute(version); err != nil {
		os.Exit(1)
	}
}
