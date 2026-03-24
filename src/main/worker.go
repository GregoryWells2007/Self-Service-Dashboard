package main

import (
	"time"

	"astraltech.xyz/accountmanager/src/logging"
)

func createWorker(interval time.Duration, task func()) {
	logging.Debugf("Creating worker that runs on a %s interval", interval.String())
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			task()
		}
	}()
}
