//go:build notray

package main

import (
	"fmt"
	"os"
)

func runAutostart(_ string, _ []string) {
	fmt.Fprintln(os.Stderr, "autostart non disponibile in questa build (compilata con -tags notray)")
	os.Exit(1)
}
