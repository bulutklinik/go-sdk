package bulutklinik

import "sync"

// TokenStore is a pluggable source for the partner token.
//
// The token is read on every request, so pointing this at a file, cache,
// database or secret manager lets a long-running process pick up a newly issued
// token without being rebuilt. An empty string means "no token"; the transport
// then fails before dispatching rather than sending an anonymous request.
//
// Implementations must be safe for concurrent use.
type TokenStore interface {
	Token() string
	SetToken(token string)
	Clear()
}

// InMemoryTokenStore is the default, concurrency-safe in-memory token store.
type InMemoryTokenStore struct {
	mu    sync.RWMutex
	token string
}

// NewInMemoryTokenStore returns a store optionally seeded with a token.
func NewInMemoryTokenStore(token string) *InMemoryTokenStore {
	return &InMemoryTokenStore{token: token}
}

func (s *InMemoryTokenStore) Token() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.token
}

func (s *InMemoryTokenStore) SetToken(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.token = token
}

func (s *InMemoryTokenStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.token = ""
}
