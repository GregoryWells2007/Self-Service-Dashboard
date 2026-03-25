package logging

import (
	"fmt"
	"log"
)

type EventType int

const (
	ReadFile EventType = iota
	AuthenticateUser
)

func Info(message string) {
	log.Printf("Info: %s", message)
}

func Infof(message string, v ...any) {
	log.Printf("Info: %s", fmt.Sprintf(message, v...))
}

func Debug(message string) {
	log.Printf("Debug: %s", message)
}

func Debugf(message string, v ...any) {
	log.Printf("Debug: %s", fmt.Sprintf(message, v...))
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
	case AuthenticateUser:
		{
			log.Printf("Authenticating user %s", eventData[0])
			break
		}
	}
}
