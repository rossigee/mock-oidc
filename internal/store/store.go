package store

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rossigee/mock-oidc/internal/config"
	"github.com/rossigee/mock-oidc/internal/crypto"
)

type authCode struct {
	userSub             string
	clientID            string
	redirectURI         string
	nonce               string
	codeChallenge       string
	codeChallengeMethod string
	expiresAt           time.Time
}

type accessToken struct {
	expiresAt time.Time
}

type refreshToken struct {
	userSub   string
	clientID  string
	expiresAt time.Time
}

// TokenInfo is returned by ListActiveTokens for test assertion purposes.
type TokenInfo struct {
	JTI       string    `json:"jti"`
	ExpiresAt time.Time `json:"expires_at"`
}

type Store struct {
	mu              sync.RWMutex
	authCodes       map[string]*authCode
	accessTokens    map[string]*accessToken // jti -> metadata (for revocation tracking)
	revokedJTIs     map[string]bool
	refreshTokens   map[string]*refreshToken
	users           map[string]*config.User
	usersByUsername map[string]*config.User
	clients         map[string]*config.Client
	keyMgr          *crypto.KeyManager
	issuer          string
}

func NewStore(cfg *config.Config, km *crypto.KeyManager, issuer string) *Store {
	s := &Store{
		authCodes:       make(map[string]*authCode),
		accessTokens:    make(map[string]*accessToken),
		revokedJTIs:     make(map[string]bool),
		refreshTokens:   make(map[string]*refreshToken),
		users:           make(map[string]*config.User),
		usersByUsername: make(map[string]*config.User),
		clients:         make(map[string]*config.Client),
		keyMgr:          km,
		issuer:          issuer,
	}

	for i := range cfg.Users {
		u := &cfg.Users[i]
		s.users[u.Sub] = u
		s.usersByUsername[u.Username] = u
	}

	for i := range cfg.Clients {
		c := &cfg.Clients[i]
		s.clients[c.ID] = c
	}

	return s
}

func (s *Store) StoreAuthCode(userSub, clientID, redirectURI, nonce, codeChallenge, codeChallengeMethod string) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	code := uuid.New().String()
	s.authCodes[code] = &authCode{
		userSub:             userSub,
		clientID:            clientID,
		redirectURI:         redirectURI,
		nonce:               nonce,
		codeChallenge:       codeChallenge,
		codeChallengeMethod: codeChallengeMethod,
		expiresAt:           time.Now().Add(5 * time.Minute),
	}

	return code
}

func (s *Store) ExchangeAuthCode(code, codeVerifier string) (userSub, clientID, redirectURI, nonce string, ok bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ac, exists := s.authCodes[code]
	if !exists || time.Now().After(ac.expiresAt) {
		delete(s.authCodes, code)
		return "", "", "", "", false
	}
	delete(s.authCodes, code)

	if ac.codeChallenge != "" {
		if codeVerifier == "" {
			return "", "", "", "", false
		}
		switch ac.codeChallengeMethod {
		case "S256":
			h := sha256.Sum256([]byte(codeVerifier))
			if base64.RawURLEncoding.EncodeToString(h[:]) != ac.codeChallenge {
				return "", "", "", "", false
			}
		default: // plain
			if codeVerifier != ac.codeChallenge {
				return "", "", "", "", false
			}
		}
	}

	return ac.userSub, ac.clientID, ac.redirectURI, ac.nonce, true
}

// IssueAccessToken signs a JWT access token and records it for revocation tracking.
func (s *Store) IssueAccessToken(userSub, clientID string, user *config.User) (string, error) {
	tokenString, jti, err := s.keyMgr.SignAccessToken(s.issuer, clientID, userSub, user.Groups, user.IsAdmin)
	if err != nil {
		return "", err
	}

	s.mu.Lock()
	s.accessTokens[jti] = &accessToken{expiresAt: time.Now().Add(time.Hour)}
	s.mu.Unlock()

	return tokenString, nil
}

// ValidateAccessToken verifies a JWT access token: signature, expiry, and presence in the
// active-token registry. Requiring JTI presence means /admin/reset immediately invalidates
// all outstanding tokens, which is important for test-suite isolation.
func (s *Store) ValidateAccessToken(tokenString string) (userSub string, ok bool) {
	claims, err := s.keyMgr.VerifyToken(tokenString)
	if err != nil {
		return "", false
	}

	if time.Now().Unix() > claims.Exp {
		return "", false
	}

	s.mu.RLock()
	at, active := s.accessTokens[claims.JTI]
	revoked := s.revokedJTIs[claims.JTI]
	s.mu.RUnlock()

	if !active || revoked || time.Now().After(at.expiresAt) {
		return "", false
	}

	return claims.Sub, true
}

// RevokeAccessToken marks a JWT's JTI as revoked.
func (s *Store) RevokeAccessToken(tokenString string) error {
	claims, err := s.keyMgr.VerifyToken(tokenString)
	if err != nil {
		return fmt.Errorf("invalid token: %w", err)
	}

	s.mu.Lock()
	s.revokedJTIs[claims.JTI] = true
	delete(s.accessTokens, claims.JTI)
	s.mu.Unlock()

	return nil
}

func (s *Store) StoreRefreshToken(userSub, clientID string) string {
	token := uuid.New().String()

	s.mu.Lock()
	s.refreshTokens[token] = &refreshToken{
		userSub:   userSub,
		clientID:  clientID,
		expiresAt: time.Now().Add(30 * 24 * time.Hour),
	}
	s.mu.Unlock()

	return token
}

// ExchangeRefreshToken validates and rotates a refresh token (single-use).
func (s *Store) ExchangeRefreshToken(token string) (userSub, clientID string, ok bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rt, exists := s.refreshTokens[token]
	if !exists || time.Now().After(rt.expiresAt) {
		delete(s.refreshTokens, token)
		return "", "", false
	}
	delete(s.refreshTokens, token)
	return rt.userSub, rt.clientID, true
}

func (s *Store) RevokeRefreshToken(token string) {
	s.mu.Lock()
	delete(s.refreshTokens, token)
	s.mu.Unlock()
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

// Admin operations

func (s *Store) AddUser(user config.User) {
	s.mu.Lock()
	defer s.mu.Unlock()

	u := user
	s.users[u.Sub] = &u
	s.usersByUsername[u.Username] = &u
}

func (s *Store) DeleteUser(sub string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	user, ok := s.users[sub]
	if !ok {
		return false
	}
	delete(s.usersByUsername, user.Username)
	delete(s.users, sub)
	return true
}

func (s *Store) AddClient(client config.Client) {
	s.mu.Lock()
	defer s.mu.Unlock()

	c := client
	s.clients[c.ID] = &c
}

func (s *Store) DeleteClient(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.clients[id]; !ok {
		return false
	}
	delete(s.clients, id)
	return true
}

// Reset flushes all tokens and auth codes, leaving users and clients intact.
func (s *Store) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.authCodes = make(map[string]*authCode)
	s.accessTokens = make(map[string]*accessToken)
	s.revokedJTIs = make(map[string]bool)
	s.refreshTokens = make(map[string]*refreshToken)
}

func (s *Store) ListActiveTokens() []TokenInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	now := time.Now()
	result := make([]TokenInfo, 0, len(s.accessTokens))
	for jti, at := range s.accessTokens {
		if now.Before(at.expiresAt) && !s.revokedJTIs[jti] {
			result = append(result, TokenInfo{JTI: jti, ExpiresAt: at.expiresAt})
		}
	}
	return result
}

func (s *Store) ListUsers() []config.User {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]config.User, 0, len(s.users))
	for _, u := range s.users {
		result = append(result, *u)
	}
	return result
}

func (s *Store) ListClients() []config.Client {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]config.Client, 0, len(s.clients))
	for _, c := range s.clients {
		result = append(result, *c)
	}
	return result
}
