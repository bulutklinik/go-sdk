package bulutklinik

import "sync"

// TokenStore is pluggable token persistence. The default is in-memory; provide
// your own implementation to persist tokens to a file, cache or database. An
// empty string means "no token".
type TokenStore interface {
	AccessToken() string
	RefreshToken() string
	SetTokens(access, refresh string)
	Clear()
}

// InMemoryTokenStore is the default, concurrency-safe in-memory token store.
type InMemoryTokenStore struct {
	mu      sync.RWMutex
	access  string
	refresh string
}

// NewInMemoryTokenStore returns a store optionally seeded with tokens.
func NewInMemoryTokenStore(access, refresh string) *InMemoryTokenStore {
	return &InMemoryTokenStore{access: access, refresh: refresh}
}

func (s *InMemoryTokenStore) AccessToken() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.access
}

func (s *InMemoryTokenStore) RefreshToken() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.refresh
}

func (s *InMemoryTokenStore) SetTokens(access, refresh string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.access = access
	s.refresh = refresh
}

func (s *InMemoryTokenStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.access = ""
	s.refresh = ""
}
