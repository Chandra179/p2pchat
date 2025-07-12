package privatechat

import (
	"fmt"
	"sync"

	"github.com/status-im/doubleratchet"
)

// InMemorySessionStorage implements doubleratchet.SessionStorage
type InMemorySessionStorage struct {
	sessions map[string]*doubleratchet.State
	mu       sync.RWMutex
}

// NewInMemorySessionStorage creates a new in-memory session storage
func NewInMemorySessionStorage() *InMemorySessionStorage {
	return &InMemorySessionStorage{
		sessions: make(map[string]*doubleratchet.State),
	}
}

// Save stores the session state
func (s *InMemorySessionStorage) Save(id []byte, state *doubleratchet.State) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[string(id)] = state
	return nil
}

// Load retrieves the session state
func (s *InMemorySessionStorage) Load(id []byte) (*doubleratchet.State, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if state, exists := s.sessions[string(id)]; exists {
		return state, nil
	}
	return nil, fmt.Errorf("session not found")
}
