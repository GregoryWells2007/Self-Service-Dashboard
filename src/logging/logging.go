package logging

import "log"

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
