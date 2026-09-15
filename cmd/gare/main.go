package main

import (
	"os"
)

const appVersion = "0.2.0"

func main() {
	if err := Execute(appVersion); err != nil {
		os.Exit(1)
	}
}
