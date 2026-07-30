package bulutklinik

import "sync"

// TokenStore is a pluggable source for the partner access token.
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

// RefreshTokenStore is an optional extension: a store that also persists the
// refresh token.
//
// Implementing it is not required — a [TokenStore] written against spec 1.0.x
// keeps working. When the injected store does not implement it, the SDK holds
// the refresh token in memory for the client's lifetime; the only consequence is
// that a process restart needs Auth.Connect rather than Auth.Refresh.
type RefreshTokenStore interface {
	TokenStore
	RefreshToken() string
	SetRefreshToken(token string)
}

// InMemoryTokenStore is the default, concurrency-safe in-memory token store.
type InMemoryTokenStore struct {
	mu      sync.RWMutex
	token   string
	refresh string
}

// NewInMemoryTokenStore returns a store optionally seeded with an access token.
// Use [NewInMemoryTokenStorePair] to seed both.
func NewInMemoryTokenStore(token string) *InMemoryTokenStore {
	return &InMemoryTokenStore{token: token}
}

// NewInMemoryTokenStorePair returns a store seeded with both tokens.
func NewInMemoryTokenStorePair(token, refresh string) *InMemoryTokenStore {
	return &InMemoryTokenStore{token: token, refresh: refresh}
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

func (s *InMemoryTokenStore) RefreshToken() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.refresh
}

func (s *InMemoryTokenStore) SetRefreshToken(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.refresh = token
}

func (s *InMemoryTokenStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.token = ""
	s.refresh = ""
}
