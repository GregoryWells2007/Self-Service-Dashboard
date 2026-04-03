package session

import (
	"net/http"
	"sync"
	"time"

	"astraltech.xyz/accountmanager/src/logging"
)

const SessionCookieName = "session_token"

type SessionManager struct {
	store SessionStore
}

var instance *SessionManager
var once sync.Once

type StoreType int

const (
	InMemory StoreType = iota
)

func GetSessionManager() *SessionManager {
	once.Do(func() {
		instance = &SessionManager{}
	})
	return instance
}

func (manager *SessionManager) SetStoreType(storeType StoreType) {
	logging.Infof("Changing session manager store type")
	switch storeType {
	case InMemory:
		{
			manager.store = NewMemoryStore()
			break
		}
	}
}

func (manager *SessionManager) CreateSession(userID string) (cookie *http.Cookie, err error) {
	logging.Debugf("Creating a new session for %s", userID)
	token, err := GenerateSessionToken(32) // Use crypto/rand for this
	if err != nil {
		return nil, err
	}
	CSRFToken, err := GenerateSessionToken(32)
	if err != nil {
		return nil, err
	}
	newSessionData := SessionData{
		UserID:    userID,
		CSRFToken: CSRFToken,
		ExpiresAt: time.Now().Add(time.Hour),
	}
	err = manager.store.Create(token, &newSessionData)
	if err != nil {
		return nil, err
	}

	newCookie := &http.Cookie{
		Name:     SessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true, // Essential: prevents JS access
		Secure:   true, // Set to TRUE in production (HTTPS)
		SameSite: http.SameSiteLaxMode,
		MaxAge:   3600, // 1 hour
	}
	return newCookie, nil
}

func (manager *SessionManager) GetSession(r *http.Request) (*SessionData, error) {
	logging.Debug("Validating session from request")
	cookie, err := r.Cookie(SessionCookieName)
	if err != nil {
		return nil, ErrSessionNotFound
	}
	token := cookie.Value
	if token == "" {
		return nil, ErrSessionNotFound
	}
	data, err := manager.store.Get(token)
	if err != nil {
		return nil, ErrSessionNotFound
	}
	return data, nil
}

func (manager *SessionManager) DeleteSession(sessionId string) error {
	return manager.store.Delete(sessionId)
}
