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

type LogLevel int

const (
	InfoLevel LogLevel = iota
	EventLevel
	DebugLevel
	WarnLevel
	ErrorLevel
	FatalLevel
)

var (
	currentLevel LogLevel = InfoLevel
)

func Info(message string) {
	if currentLevel > InfoLevel {
		return
	}
	log.Printf("Info: %s", message)
}

func Infof(message string, v ...any) {
	if currentLevel > InfoLevel {
		return
	}
	log.Printf("Info: %s", fmt.Sprintf(message, v...))
}

func Debug(message string) {
	if currentLevel > DebugLevel {
		return
	}
	log.Printf("Debug: %s", message)
}

func Debugf(message string, v ...any) {
	if currentLevel > DebugLevel {
		return
	}
	log.Printf("Debug: %s", fmt.Sprintf(message, v...))
}

func Warn(message string) {
	if currentLevel > WarnLevel {
		return
	}
	log.Printf("Warn: %s", message)
}

func Warnf(message string, v ...any) {
	if currentLevel > WarnLevel {
		return
	}
	log.Printf("Warn: %s", fmt.Sprintf(message, v...))
}

func Error(message string) {
	if currentLevel > ErrorLevel {
		return
	}
	log.Printf("Error: %s", message)
}

func Errorf(message string, v ...any) {
	if currentLevel > ErrorLevel {
		return
	}
	log.Printf("Error: %s", fmt.Sprintf(message, v...))
}

func Fatal(message string) {
	if currentLevel > FatalLevel {
		return
	}
	log.Fatal(message)
}

func Fatalf(message string, v ...any) {
	if currentLevel > FatalLevel {
		return
	}
	log.Fatalf(message, v...)
}

func Event(eventType EventType, eventData ...any) {
	if currentLevel > EventLevel {
		return
	}

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

func SetLogLovel(level LogLevel) {
	currentLevel = level
}
