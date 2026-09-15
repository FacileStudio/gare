package main

import (
	"os"
)

var version = "0.4.2"

func main() {
	if err := Execute(version); err != nil {
		os.Exit(1)
	}
}
