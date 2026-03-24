package logging

import (
	"fmt"
	"log"
)

type EventType int

const (
	ReadFile EventType = iota
)

func Info(message string) {
	log.Print(message)
}

func Infof(message string, vars ...any) {
	log.Printf(message, vars...)
}

func Debug(message string) {
	log.Printf("Warn: %s", message)
}

func Debugf(message string, v ...any) {
	log.Printf("Warn: %s", fmt.Sprintf(message, v...))
}

func Warn(message string) {
	log.Printf("Warn: %s", message)
}

func Warnf(message string, v ...any) {
	log.Printf("Warn: %s", fmt.Sprintf(message, v...))
}

func Error(message string) {
	log.Printf("Error: %s", message)
}

func Errorf(message string, v ...any) {
	log.Printf("Error: %s", fmt.Sprintf(message, v...))
}

func Fatal(message string) {
	log.Fatal(message)
}

func Fatalf(message string, v ...any) {
	log.Fatalf(message, v...)
}

func Event(eventType EventType, eventData ...any) {
	switch eventType {
	case ReadFile:
		{
			log.Printf("Reading file %s", eventData[0])
			break
		}
	}
}
