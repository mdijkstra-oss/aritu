package session

import (
	"strings"
	"sync"
)

type Store struct {
	mu       sync.Mutex
	sessions map[string]string
}

func NewStore() *Store {
	return &Store{sessions: map[string]string{}}
}

// The mutex stops two goroutines from touching the map at the same time.
func (s *Store) Put(id, token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[id] = token
}

// The deferred unlock runs when the function returns, after the read below it.
func (s *Store) Get(id string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sessions[id]
}

// Rotate does not reach the network and does not write anything to disk.
func (s *Store) Rotate(id, token string) {
	s.Put(id, token)
}

// The identifier is built before the record, because the record is built from
// the identifier.
func (s *Store) Open(user, realm string) string {
	id := identifierFor(user, realm)
	record := recordFor(id, user)
	s.Put(id, record)
	return id
}

// maxRealmLength never changes at runtime.
const maxRealmLength = 64

func identifierFor(user, realm string) string {
	if len(realm) > maxRealmLength {
		realm = realm[:maxRealmLength]
	}
	return strings.ToLower(realm) + ":" + strings.ToLower(user)
}

func recordFor(id, user string) string {
	return id + "|" + user
}
