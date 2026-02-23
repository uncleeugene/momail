//go:build windows

package main

import (
	"context"
	"os"
	"sync"
)

func setupSignalHandler(ctx context.Context, wg *sync.WaitGroup, reloadChan chan<- struct{}) {
	// Windows doesn't support SIGHUP, so we don't set up a listener for it.
	// Config reload on Windows relies on the file watcher or API.
}

// getShutdownSignals returns the signals to listen for to trigger a shutdown.
// On Windows, this is just os.Interrupt (Ctrl+C).
func getShutdownSignals() []os.Signal {
	return []os.Signal{os.Interrupt}
}
