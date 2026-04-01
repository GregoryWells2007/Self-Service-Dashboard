package session

import (
	"errors"
	"sync"
	"time"

	"astraltech.xyz/accountmanager/src/logging"
	"astraltech.xyz/accountmanager/src/worker"
)

var ErrSessionNotFound = errors.New("session not found")
var ErrSessionAlreadyExists = errors.New("session already exists")
var ErrSessionExpired = errors.New("session expired")

type MemoryStore struct {
	sessions map[string]*SessionData
	lock     sync.RWMutex
}

func NewMemoryStore() *MemoryStore {
	logging.Debug("Creating new in memory session store")
	store := &MemoryStore{
		sessions: make(map[string]*SessionData),
	}
	worker.CreateWorker(time.Minute*5, store.cleanup)
	return store
}

func (m *MemoryStore) Create(sessionID string, session *SessionData) (err error) {
	hashedSession := hashSession(sessionID)

	m.lock.Lock()
	defer m.lock.Unlock()
	_, exist := m.sessions[hashedSession]
	if exist {
		return ErrSessionAlreadyExists
	}

	m.sessions[hashedSession] = session
	return nil
}
func (m *MemoryStore) Get(sessionID string) (*SessionData, error) {
	m.lock.RLock()
	hashed := hashSession(sessionID)
	data, exists := m.sessions[hashed]
	m.lock.RUnlock()
	if exists == false {
		return nil, ErrSessionNotFound
	}
	if time.Now().After(data.ExpiresAt) {
		_ = m.Delete(sessionID) // ignore error
		return nil, ErrSessionExpired
	}
	copy := *data
	return &copy, nil
}
func (m *MemoryStore) Update(sessionID string, session *SessionData) error {
	hashedSession := hashSession(sessionID)

	m.lock.Lock()
	defer m.lock.Unlock()
	_, exist := m.sessions[hashedSession]
	if !exist {
		return ErrSessionNotFound
	}
	m.sessions[hashedSession] = session
	return nil
}

func (m *MemoryStore) cleanup() {
	logging.Debug("Cleaning up memory store sessions")
	now := time.Now()

	m.lock.Lock()
	defer m.lock.Unlock()

	deleted := 0
	for id, session := range m.sessions {
		if now.After(session.ExpiresAt) {
			delete(m.sessions, id)
			deleted = deleted + 1
		}
	}
	logging.Infof("Cleaned up %d stale sessions", deleted)
}

func (m *MemoryStore) Delete(sessionID string) error {
	hashedSession := hashSession(sessionID)

	m.lock.Lock()
	defer m.lock.Unlock()
	_, exist := m.sessions[hashedSession]
	if !exist {
		return ErrSessionNotFound
	}
	delete(m.sessions, hashedSession)
	return nil
}
