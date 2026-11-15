package bot

import (
	"sync"
	"time"
)

// SessionStore — потокобезопасное хранилище сессий пользователей в памяти.
type SessionStore struct {
	mu       sync.RWMutex
	sessions map[int64]*UserSession
}

// NewSessionStore создаёт новый экземпляр хранилища.
func NewSessionStore() *SessionStore {
	return &SessionStore{
		sessions: make(map[int64]*UserSession),
	}
}

// Get возвращает сессию пользователя или создаёт новую, если её нет.
func (s *SessionStore) Get(chatID int64) *UserSession {
	s.mu.RLock()
	session, exists := s.sessions[chatID]
	s.mu.RUnlock()
	if exists {
		return session
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if session, exists = s.sessions[chatID]; exists {
		return session
	}
	session = NewUserSession(chatID)
	s.sessions[chatID] = session
	return session
}

// Set сохраняет сессию пользователя.
func (s *SessionStore) Set(session *UserSession) {
	s.mu.Lock()
	session.UpdatedAt = time.Now()
	s.sessions[session.ChatID] = session
	s.mu.Unlock()
}

// Delete удаляет сессию пользователя.
func (s *SessionStore) Delete(chatID int64) {
	s.mu.Lock()
	delete(s.sessions, chatID)
	s.mu.Unlock()
}

// Count возвращает количество активных сессий.
func (s *SessionStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.sessions)
}

// CleanupRemoves сессии старше указанного времени (для предотвращения утечки памяти).
func (s *SessionStore) Cleanup(maxAge time.Duration) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	count := 0
	for chatID, session := range s.sessions {
		if now.Sub(session.UpdatedAt) > maxAge {
			delete(s.sessions, chatID)
			count++
		}
	}
	return count
}
