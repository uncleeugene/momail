package cron

import (
	"log"
	"os/exec"

	"github.com/robfig/cron/v3"
	"uncleeugene.kz/momail/config"
	"uncleeugene.kz/momail/logutil"
)

// Start initializes and starts the cron scheduler with tasks from the config.
func Start(cfg *config.Config) (shutdown func()) {
	if len(cfg.Tasks) == 0 {
		return func() {} // Return a no-op shutdown function
	}

	c := cron.New()

	for _, task := range cfg.Tasks {
		// Capture task in a local variable for the closure.
		t := task
		_, err := c.AddFunc(t.CronSpec, func() {
			log.Println(logutil.Debug("Cron: executing task '%s'", t.Command))
			cmd := exec.Command("sh", "-c", t.Command)
			if output, err := cmd.CombinedOutput(); err != nil {
				log.Println(logutil.Error("Cron task execution failed: %v\nOutput: %s", err, string(output)))
			}
		})
		if err != nil {
			log.Println(logutil.Error("Error scheduling task '%s' with spec '%s': %v", t.Command, t.CronSpec, err))
		}
	}

	log.Println(logutil.Debug("Starting cron scheduler with %d tasks.", len(c.Entries())))
	c.Start()

	return func() {
		<-c.Stop().Done()
	}
}
