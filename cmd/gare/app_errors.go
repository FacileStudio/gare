package main

import "fmt"

// appConfigError wraps a failed application config lookup with a remedy.
func appConfigError(name string, err error) error {
	return fmt.Errorf("app %q not found — run `gare list` to see managed apps: %w", name, err)
}
