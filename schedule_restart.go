package main

import (
	"os"
	"time"

	log "github.com/Sirupsen/logrus"
)

func scheduleRestart() {
	log.WithFields(log.Fields{"interval": RestartTimeout.String()}).Info("Scheduled Restart Activated")

	time.AfterFunc(RestartTimeout, func() {
		log.Info("Restarting on Schedule")
		os.Exit(0)
	})
}
