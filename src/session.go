package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"log"
	"net/http"
	"sync"
	"time"
)

type SessionData struct {
	loggedIn    bool
	data        *UserData
	timeCreated time.Time
	CSRFToken   string
}

var (
	sessions     = make(map[string]SessionData)
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
	token, err := GenerateSessionToken(32) // Use crypto/rand for this
	if err != nil {
		log.Print(err)
		return nil
	}
	CSRFToken, err := GenerateSessionToken(32)
	if err != nil {
		log.Print(err)
		return nil
	}

	tokenEncoded := sha256.Sum256([]byte(token))
	tokenEncodedString := string(tokenEncoded[:])

	sessionMutex.Lock()
	defer sessionMutex.Unlock()
	loggedIn := false
	if userData != nil {
		loggedIn = true
	}
	sessions[tokenEncodedString] = SessionData{
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
	cookie, err := r.Cookie("session_token")
	if err != nil {
		return false, &SessionData{}
	}
	token := cookie.Value

	tokenEncoded := sha256.Sum256([]byte(token))
	tokenEncodedString := string(tokenEncoded[:])

	sessionMutex.Lock()
	sessionData, exists := sessions[tokenEncodedString]
	sessionMutex.Unlock()
	if !exists || !sessionData.loggedIn {
		return false, &SessionData{}
	}
	return true, &sessionData
}
