package session

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
)

// helper function for secure session storage
func GenerateSessionToken(length int) (string, error) {
	b := make([]byte, length)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	token := base64.RawURLEncoding.EncodeToString(b)
	return token, nil
}

// more helper
func hashSession(session_id string) string {
	tokenEncoded := sha256.Sum256([]byte(session_id))
	return base64.RawURLEncoding.EncodeToString(tokenEncoded[:])
}
