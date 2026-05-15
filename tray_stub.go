//go:build notray

package main

import (
	"fmt"
	"os"
)

func runTray(_ *RecordConfig) {
	fmt.Fprintln(os.Stderr, "questa build non include la tray icon (compilata con -tags notray)")
	fmt.Fprintln(os.Stderr, "usa 'call-recorder record' dalla riga di comando")
	os.Exit(1)
}
