package store

import (
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rossigee/mock-oidc/internal/config"
)

type authCode struct {
	userSub    string
	clientID   string
	expiresAt  time.Time
	redirectURI string
}

type accessToken struct {
	userSub   string
	expiresAt time.Time
}

type Store struct {
	mu               sync.RWMutex
	authCodes        map[string]*authCode
	accessTokens     map[string]*accessToken
	users            map[string]*config.User
	usersByUsername  map[string]*config.User
	clients          map[string]*config.Client
}

func NewStore(cfg *config.Config) *Store {
	s := &Store{
		authCodes:       make(map[string]*authCode),
		accessTokens:    make(map[string]*accessToken),
		users:           make(map[string]*config.User),
		usersByUsername: make(map[string]*config.User),
		clients:         make(map[string]*config.Client),
	}

	// Load users
	for i := range cfg.Users {
		user := &cfg.Users[i]
		s.users[user.Sub] = user
		s.usersByUsername[user.Username] = user
	}

	// Load clients
	for i := range cfg.Clients {
		client := &cfg.Clients[i]
		s.clients[client.ID] = client
	}

	return s
}

func (s *Store) StoreAuthCode(userSub, clientID, redirectURI string) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	code := uuid.New().String()
	s.authCodes[code] = &authCode{
		userSub:    userSub,
		clientID:   clientID,
		expiresAt:  time.Now().Add(5 * time.Minute),
		redirectURI: redirectURI,
	}

	return code
}

func (s *Store) ExchangeAuthCode(code string) (userSub, clientID, redirectURI string, ok bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ac, exists := s.authCodes[code]
	if !exists {
		return "", "", "", false
	}

	if time.Now().After(ac.expiresAt) {
		delete(s.authCodes, code)
		return "", "", "", false
	}

	defer delete(s.authCodes, code)
	return ac.userSub, ac.clientID, ac.redirectURI, true
}

func (s *Store) StoreAccessToken(userSub string) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	token := uuid.New().String()
	s.accessTokens[token] = &accessToken{
		userSub:   userSub,
		expiresAt: time.Now().Add(1 * time.Hour),
	}

	return token
}

func (s *Store) ValidateAccessToken(token string) (userSub string, ok bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	at, exists := s.accessTokens[token]
	if !exists {
		return "", false
	}

	if time.Now().After(at.expiresAt) {
		return "", false
	}

	return at.userSub, true
}

func (s *Store) GetUserBySub(sub string) (*config.User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, ok := s.users[sub]
	return user, ok
}

func (s *Store) GetUserByUsername(username string) (*config.User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, ok := s.usersByUsername[username]
	return user, ok
}

func (s *Store) GetClient(clientID string) (*config.Client, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	client, ok := s.clients[clientID]
	return client, ok
}
