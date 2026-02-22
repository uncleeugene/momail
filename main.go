package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/acarl005/stripansi"
	"github.com/fsnotify/fsnotify"
	"gopkg.in/yaml.v3"
	"uncleeugene.kz/momail/api"
	"uncleeugene.kz/momail/binkp"
	"uncleeugene.kz/momail/config"
	"uncleeugene.kz/momail/cron"
	"uncleeugene.kz/momail/ftn"
	"uncleeugene.kz/momail/logstream"
	"uncleeugene.kz/momail/logutil"
	"uncleeugene.kz/momail/monitor"
	"uncleeugene.kz/momail/scheduler"

	"github.com/urfave/cli/v2"
)

// ansiStrippingWriter is an io.Writer that strips ANSI escape codes
// before writing to the underlying writer.
type ansiStrippingWriter struct {
	writer io.Writer
}

// Write strips ANSI codes and writes to the underlying writer.
func (w *ansiStrippingWriter) Write(p []byte) (n int, err error) {
	stripped := stripansi.Strip(string(p))
	if _, err := w.writer.Write([]byte(stripped)); err != nil {
		return 0, err
	}
	return len(p), nil
}

func main() {
	app := &cli.App{
		Name:  "momail",
		Usage: "A modern FidoNet Technology Network (FTN) mailer",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Value:   "config.yaml",
				Usage:   "Load configuration from `FILE`",
			},
			&cli.StringFlag{
				Name:    "poll",
				Aliases: []string{"p"},
				Usage:   "Poll a specific node address (e.g. -p 2:5020/828)",
			},
			&cli.StringFlag{
				Name:    "queue-poll",
				Aliases: []string{"q"},
				Usage:   "Queue a poll for an `ADDRESS` by creating a poll file",
			},
			&cli.BoolFlag{
				Name:  "crash",
				Usage: "Create a crash poll (.clo) instead of a normal poll (.flo)",
			},
			&cli.BoolFlag{
				Name:  "hold",
				Usage: "Create a hold poll (.hlo) instead of a normal poll (.flo)",
			},
			&cli.BoolFlag{
				Name:  "direct",
				Usage: "Create a direct poll (.dlo) instead of a crash poll (.clo)",
			},
			&cli.BoolFlag{
				Name:  "cut-logs",
				Usage: "Cut log files to the size specified in config and exit",
			},

			&cli.BoolFlag{
				Name:    "watch-config",
				Aliases: []string{"w"},
				Usage:   "Automatically reload config on change (daemon mode only)",
			},
		},
		Action: func(c *cli.Context) error {
			cfgPath := c.String("config")
			cfg, err := config.Load(cfgPath)
			if err != nil {
				return err
			}

			// Handle log cutting command and exit
			if c.Bool("cut-logs") {
				// Use a simple logger for this one-off command
				log.SetOutput(os.Stdout)
				log.SetFlags(0)
				if err := cutLogs(cfg); err != nil {
					// Use fmt to print to stderr since logging is basic
					fmt.Fprintf(os.Stderr, "Error cutting logs: %v\n", err)
					return err
				}
				return nil
			}

			monitor.SetNodeAddress(cfg.ParsedAddress.String())

			// Setup Logging
			logFile, err := os.OpenFile(cfg.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				return fmt.Errorf("failed to open log file: %w", err)
			}
			defer logFile.Close()

			// Start the log streaming hub
			go logstream.LogHub.Run()
			log.SetOutput(io.MultiWriter(os.Stdout, &ansiStrippingWriter{writer: logFile}, logstream.LogHub))

			// Handle one-off commands that should exit immediately
			isOneOffCommand := c.IsSet("queue-poll") || c.IsSet("poll")
			if isOneOffCommand {
				// Fall through to the command handlers below
			} else {
				// This is daemon mode
				return runDaemon(c, cfg, cfgPath)
			}

			// Handle Queue Poll Command
			if queueAddr := c.String("queue-poll"); queueAddr != "" {
				target, err := ftn.ParseFidoAddress(queueAddr, cfg.ParsedAddress.Zone)
				if err != nil {
					return fmt.Errorf("invalid queue address: %w", err)
				}

				isCrash := c.Bool("crash")
				isHold := c.Bool("hold")
				isDirect := c.Bool("direct")
				flavor := "normal"
				if isCrash {
					flavor = "crash"
				} else if isHold {
					flavor = "hold"
				} else if isDirect {
					flavor = "direct"
				}

				pollPath, err := scheduler.GetPollFilePath(cfg, target, flavor)
				if err != nil {
					return fmt.Errorf("could not determine poll file path: %w", err)
				}

				log.Println(logutil.Info("Creating poll file: %s", pollPath))
				f, err := os.Create(pollPath)
				if err != nil {
					return fmt.Errorf("failed to create poll file: %w", err)
				}
				f.Close()

				log.Println(logutil.Success("Successfully queued %s poll for %s", flavor, target))
				return nil
			}

			// Handle Poll Command
			if pollAddr := c.String("poll"); pollAddr != "" {
				target, err := ftn.ParseFidoAddress(pollAddr, cfg.ParsedAddress.Zone)
				if err != nil {
					return fmt.Errorf("invalid poll address: %w", err)
				}

				var link *config.Link
				for i := range cfg.Links {
					l := &cfg.Links[i]
					if l.ParsedAddress.Zone == target.Zone &&
						l.ParsedAddress.Net == target.Net &&
						l.ParsedAddress.Node == target.Node &&
						l.ParsedAddress.Point == target.Point {
						link = l
						break
					}
				}

				if link == nil {
					return fmt.Errorf("no link configuration found for %s", target)
				}

				return binkp.Dial(cfg, link)
			}

			return nil
		},
	}

	// Use a context that is cancelled on SIGINT/SIGTERM
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	err := app.RunContext(ctx, os.Args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	}
}

func cutLogs(cfg *config.Config) error {
	if cfg.LogMaxSize <= 0 {
		log.Println("log_max_size is not configured or is zero, skipping cut.")
		return nil
	}

	maxSizeBytes := int64(cfg.LogMaxSize * 1024)

	if err := cutLogFile(cfg.LogFile, maxSizeBytes); err != nil {
		return fmt.Errorf("failed to cut main log file: %w", err)
	}

	if cfg.SessionLog != "" {
		if err := cutLogFile(cfg.SessionLog, maxSizeBytes); err != nil {
			return fmt.Errorf("failed to cut session log file: %w", err)
		}
	}

	log.Println("Log cutting complete.")
	return nil
}

func cutLogFile(path string, maxSize int64) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // File doesn't exist, nothing to do.
		}
		return err
	}

	if info.Size() <= maxSize {
		return nil // File is smaller than the limit.
	}

	log.Printf("Cutting %s (size: %d KB) to %d KB...", path, info.Size()/1024, maxSize/1024)

	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err := file.Seek(info.Size()-maxSize, io.SeekStart); err != nil {
		return err
	}

	reader := bufio.NewReader(file)
	// Read and discard the first (potentially partial) line to align to a newline
	if _, err := reader.ReadBytes('\n'); err != nil && err != io.EOF {
		return err
	}

	tempPath := path + ".tmp"
	tempFile, err := os.Create(tempPath)
	if err != nil {
		return err
	}

	_, copyErr := io.Copy(tempFile, reader)
	closeErr := tempFile.Close()

	if copyErr != nil {
		os.Remove(tempPath)
		return copyErr
	}
	if closeErr != nil {
		os.Remove(tempPath)
		return closeErr
	}

	return os.Rename(tempPath, path)
}

func runDaemon(c *cli.Context, cfg *config.Config, cfgPath string) error {
	var wg sync.WaitGroup
	ctx := c.Context

	// Channel to signal config reload
	reloadChan := make(chan struct{}, 1)

	if c.Bool("watch-config") {
		wg.Add(1)
		go watchConfig(ctx, &wg, cfgPath, reloadChan)
	}

	// Listen for SIGHUP to reload config
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

	isFirstRun := true
	// Main lifecycle loop
	for {
		log.Println(logutil.Success("momail starting up..."))
		log.Println(logutil.Info("-> Node Address: %s", cfg.ParsedAddress.String()))
		log.Println(logutil.Info("-> Outbound Dir: %s", cfg.Outbound))

		// Create a context for this run of services
		serviceCtx, serviceCancel := context.WithCancel(ctx)

		// Start services
		binkp.OnSessionEnd = func(success bool) {
			if success {
				scheduler.TriggerScan("session finished")
			}
		}
		schedulerShutdown := scheduler.Start(cfg, isFirstRun)
		isFirstRun = false
		cronShutdown := cron.Start(cfg)

		reloadConfig := func() {
			select {
			case reloadChan <- struct{}{}:
			default:
			}
		}
		api.Start(serviceCtx, cfg, reloadConfig)

		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := binkp.Serve(serviceCtx, cfg); err != nil {
				log.Println(logutil.Error("BinkP server exited with error: %v", err))
			}
		}()

		// Wait for reload signal or program exit
		select {
		case <-reloadChan:
			log.Println(logutil.Warn("Config file changed. Reloading..."))

			// Stop current services
			log.Println(logutil.Debug("Stopping services..."))
			schedulerShutdown()
			cronShutdown()
			serviceCancel() // This will stop the binkp server

			// Reload config
			newCfg, err := config.Load(cfgPath)
			if err != nil {
				var typeErr *yaml.TypeError
				if errors.As(err, &typeErr) {
					for _, msg := range typeErr.Errors {
						log.Println(logutil.Error("Config error: %s", msg))
					}
					log.Println(logutil.Error("Configuration reload failed. Keeping old config."))
				} else {
					log.Println(logutil.Error("Failed to reload config: %v. Keeping old config.", err))
				}
			} else {
				cfg = newCfg
				log.Println(logutil.Success("Configuration reloaded successfully."))
			}
			continue // Restart the loop
		case <-ctx.Done():
			// Program is exiting
			log.Println(logutil.Warn("Shutdown signal received. Stopping services..."))
			schedulerShutdown()
			cronShutdown()
			serviceCancel()
			goto end
		}
	}
end:
	wg.Wait()
	log.Println(logutil.Success("momail shut down gracefully."))
	return nil
}

func watchConfig(ctx context.Context, wg *sync.WaitGroup, path string, reloadChan chan<- struct{}) {
	defer wg.Done()
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Println(logutil.Error("ConfigWatcher: Failed to create watcher: %v", err))
		return
	}
	defer watcher.Close()

	// Resolve absolute path to ensure consistent comparison
	absPath, err := filepath.Abs(path)
	if err != nil {
		log.Println(logutil.Error("ConfigWatcher: Failed to resolve absolute path for %s: %v", path, err))
		return
	}

	// Watch the directory containing the file to handle editor atomic writes
	dir := filepath.Dir(absPath)
	if err := watcher.Add(dir); err != nil {
		log.Println(logutil.Error("ConfigWatcher: Failed to watch directory '%s': %v", dir, err))
		return
	}

	log.Println(logutil.Debug("ConfigWatcher: Watching %s for changes...", absPath))

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}

			eventAbsPath, _ := filepath.Abs(event.Name)
			// We only care about writes to our specific config file.
			// Editors often use Rename/Create for atomic saves, so we watch for those too.
			if eventAbsPath == absPath && (event.Has(fsnotify.Write) || event.Has(fsnotify.Create) || event.Has(fsnotify.Rename)) {
				log.Println(logutil.Debug("ConfigWatcher: Config file change detected."))
				// Debounce: non-blocking send
				select {
				case reloadChan <- struct{}{}:
				default:
				}
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Println(logutil.Error("ConfigWatcher: Error: %v", err))
		case <-ctx.Done():
			return
		}
	}
}
