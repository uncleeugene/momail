package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sync"

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

const Version = "0.1.1-alpha"

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
			&cli.BoolFlag{
				Name:    "version",
				Aliases: []string{"v"},
				Usage:   "Print version information and exit",
			},
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
				Name:    "watch-config",
				Aliases: []string{"w"},
				Usage:   "Automatically reload config on change (daemon mode only)",
			},
		},
		Action: func(c *cli.Context) error {

			if c.Bool("version") {
				fmt.Println("momail v" + Version)
				return nil
			}

			cfgPath := c.String("config")
			cfg, err := config.Load(cfgPath)
			if err != nil {
				return err
			}

			cfg.Version = Version
			monitor.SetNodeAddress(cfg.ParsedAddress.String())

			// Setup Logging
			logWriter, err := logutil.NewRotatableWriter(cfg.LogFile, cfg.LogMaxSize)
			if err != nil {
				return fmt.Errorf("failed to initialize log writer: %w", err)
			}

			// Start the log streaming hub
			go logstream.LogHub.Run()
			log.SetOutput(io.MultiWriter(os.Stdout, &ansiStrippingWriter{writer: logWriter}, logstream.LogHub))

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
	ctx, stop := signal.NotifyContext(context.Background(), getShutdownSignals()...)
	defer stop()

	err := app.RunContext(ctx, os.Args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	}
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

	setupSignalHandler(ctx, &wg, reloadChan)

	isFirstRun := true
	// Main lifecycle loop
	for {
		log.Println(logutil.Success("momail v%s starting up...", cfg.Version))
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
