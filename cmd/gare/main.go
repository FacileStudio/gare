package main

import (
	"os"
)

var version = "0.8.6"

func main() {
	if err := Execute(version); err != nil {
		os.Exit(1)
	}
}
