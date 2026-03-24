package main

import (
	"time"
)

func createWorker(interval time.Duration, task func()) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			task()
		}
	}()
}
