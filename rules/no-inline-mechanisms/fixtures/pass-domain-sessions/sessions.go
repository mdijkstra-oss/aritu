package auth

import (
	"slices"
	"sync"
	"time"
)

type Session struct {
	User      string
	Scopes    []string
	StartedAt time.Time
}

type SessionStore struct {
	mu       sync.Mutex
	byToken  map[string]Session
	lifetime time.Duration
}

func NewSessionStore(lifetime time.Duration) *SessionStore {
	return &SessionStore{byToken: map[string]Session{}, lifetime: lifetime}
}

func (s *SessionStore) SignIn(token string, user string, scopes []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.byToken[token] = Session{User: user, Scopes: scopes, StartedAt: time.Now()}
}

func (s *SessionStore) Authorize(token string, scope string) (Session, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	session, signedIn := s.byToken[token]
	if !signedIn || time.Since(session.StartedAt) > s.lifetime {
		return Session{}, false
	}
	return session, slices.Contains(session.Scopes, scope)
}

func (s *SessionStore) SignOut(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.byToken, token)
}
