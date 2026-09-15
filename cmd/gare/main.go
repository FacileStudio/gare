package main

import (
	"os"
)

const appVersion = "0.1.0"

func main() {
	if err := Execute(appVersion); err != nil {
		os.Exit(1)
	}
}
