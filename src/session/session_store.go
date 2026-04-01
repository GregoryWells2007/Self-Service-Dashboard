package session

import "time"

type SessionData struct {
	UserID    string
	CSRFToken string
	ExpiresAt time.Time
}

type SessionStore interface {
	Create(sessionID string, session *SessionData) error
	Get(sessionID string) (*SessionData, error)
	Update(sessionID string, session *SessionData) error
	Delete(sessionID string) error
}
