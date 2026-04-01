package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"sync"
	"time"

	"astraltech.xyz/accountmanager/src/logging"
)

type SessionData struct {
	loggedIn    bool
	data        *UserData
	timeCreated time.Time
	CSRFToken   string
}

var (
	sessions     = make(map[string]*SessionData)
	sessionMutex sync.Mutex
)

func GenerateSessionToken(length int) (string, error) {
	b := make([]byte, length)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	token := base64.RawURLEncoding.EncodeToString(b)
	return token, nil
}

func createSession(userData *UserData) *http.Cookie {
	logging.Debugf("Creating a new session for %s", userData.Username)
	token, err := GenerateSessionToken(32) // Use crypto/rand for this
	if err != nil {
		logging.Error(err.Error())
		return nil
	}
	CSRFToken, err := GenerateSessionToken(32)
	if err != nil {
		logging.Error(err.Error())
		return nil
	}

	encodedToken := hashSession(token)

	sessionMutex.Lock()
	defer sessionMutex.Unlock()
	loggedIn := false
	if userData != nil {
		loggedIn = true
	}
	sessions[encodedToken] = &SessionData{
		data:        userData,
		timeCreated: time.Now(),
		CSRFToken:   CSRFToken,
		loggedIn:    loggedIn,
	}
	cookie := &http.Cookie{
		Name:     "session_token",
		Value:    token,
		Path:     "/",
		HttpOnly: true, // Essential: prevents JS access
		Secure:   true, // Set to TRUE in production (HTTPS)
		SameSite: http.SameSiteLaxMode,
		MaxAge:   3600, // 1 hour
	}
	return cookie
}

func validateSession(r *http.Request) (bool, *SessionData) {
	logging.Debugf("Validating session")
	cookie, err := r.Cookie("session_token")
	if err != nil {
		logging.Error(err.Error())
		return false, &SessionData{}
	}
	token := cookie.Value
	token = hashSession(token)

	sessionMutex.Lock()
	sessionData, exists := sessions[token]
	sessionMutex.Unlock()
	if !exists || !sessionData.loggedIn {
		return false, &SessionData{}
	}
	logging.Infof("Validated session for %s", sessionData.data.Username)
	return true, sessionData
}

func hashSession(session_id string) string {
	tokenEncoded := sha256.Sum256([]byte(session_id))
	return base64.RawURLEncoding.EncodeToString(tokenEncoded[:])
}

func deleteSession(session_id string) {
	sessionMutex.Lock()
	delete(sessions, session_id)
	sessionMutex.Unlock()
}
