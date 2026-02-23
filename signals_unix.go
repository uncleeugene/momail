//go:build !windows

package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"uncleeugene.kz/momail/logutil"
)

func setupSignalHandler(ctx context.Context, wg *sync.WaitGroup, reloadChan chan<- struct{}) {
	hupChan := make(chan os.Signal, 1)
	signal.Notify(hupChan, syscall.SIGHUP)
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer signal.Stop(hupChan)
		for {
			select {
			case <-ctx.Done():
				return
			case <-hupChan:
				log.Println(logutil.Info("Received SIGHUP. Triggering config reload..."))
				select {
				case reloadChan <- struct{}{}:
				default:
				}
			}
		}
	}()
}

// getShutdownSignals returns the signals to listen for to trigger a shutdown.
func getShutdownSignals() []os.Signal {
	return []os.Signal{os.Interrupt, syscall.SIGTERM}
}
