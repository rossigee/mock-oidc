package crypto

import (
	stdcrypto "crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/big"
	"strings"
	"time"

	"github.com/google/uuid"
)

type KeyManager struct {
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	kid        string
}

func NewKeyManager() (*KeyManager, error) {
	slog.Info("generating RSA-2048 key pair")
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate RSA key: %w", err)
	}

	return &KeyManager{
		privateKey: privateKey,
		publicKey:  &privateKey.PublicKey,
		kid:        uuid.New().String()[:8],
	}, nil
}

type JWKSResponse struct {
	Keys []JWK `json:"keys"`
}

type JWK struct {
	Kty string `json:"kty"`
	Use string `json:"use"`
	Kid string `json:"kid"`
	Alg string `json:"alg"`
	N   string `json:"n"`
	E   string `json:"e"`
}

func (km *KeyManager) JWKS() JWKSResponse {
	return JWKSResponse{
		Keys: []JWK{
			{
				Kty: "RSA",
				Use: "sig",
				Kid: km.kid,
				Alg: "RS256",
				N:   base64.RawURLEncoding.EncodeToString(km.publicKey.N.Bytes()),
				E:   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(km.publicKey.E)).Bytes()),
			},
		},
	}
}

type IDTokenClaims struct {
	Sub           string   `json:"sub"`
	Iss           string   `json:"iss"`
	Aud           string   `json:"aud"`
	Exp           int64    `json:"exp"`
	Iat           int64    `json:"iat"`
	Nonce         string   `json:"nonce,omitempty"`
	Email         string   `json:"email,omitempty"`
	EmailVerified bool     `json:"email_verified,omitempty"`
	Name          string   `json:"name,omitempty"`
	Groups        []string `json:"groups,omitempty"`
	IsAdmin       bool     `json:"is_admin,omitempty"`
}

type AccessTokenClaims struct {
	Sub     string   `json:"sub"`
	Iss     string   `json:"iss"`
	Aud     string   `json:"aud"`
	Exp     int64    `json:"exp"`
	Iat     int64    `json:"iat"`
	JTI     string   `json:"jti"`
	Groups  []string `json:"groups,omitempty"`
	IsAdmin bool     `json:"is_admin,omitempty"`
}

type jwtHeader struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
	Kid string `json:"kid"`
}

func (km *KeyManager) sign(header jwtHeader, claimsJSON []byte) (string, error) {
	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", fmt.Errorf("failed to marshal header: %w", err)
	}

	message := base64.RawURLEncoding.EncodeToString(headerJSON) + "." + base64.RawURLEncoding.EncodeToString(claimsJSON)
	hash := sha256.Sum256([]byte(message))

	sig, err := rsa.SignPKCS1v15(rand.Reader, km.privateKey, stdcrypto.SHA256, hash[:])
	if err != nil {
		return "", fmt.Errorf("failed to sign: %w", err)
	}

	return message + "." + base64.RawURLEncoding.EncodeToString(sig), nil
}

func (km *KeyManager) SignIDToken(issuer, clientID, subject, nonce, email, name string, emailVerified, isAdmin bool, groups []string) (string, error) {
	now := time.Now()
	claimsJSON, err := json.Marshal(IDTokenClaims{
		Sub:           subject,
		Iss:           issuer,
		Aud:           clientID,
		Exp:           now.Add(time.Hour).Unix(),
		Iat:           now.Unix(),
		Nonce:         nonce,
		Email:         email,
		EmailVerified: emailVerified,
		Name:          name,
		Groups:        groups,
		IsAdmin:       isAdmin,
	})
	if err != nil {
		return "", fmt.Errorf("failed to marshal claims: %w", err)
	}

	return km.sign(jwtHeader{Alg: "RS256", Typ: "JWT", Kid: km.kid}, claimsJSON)
}

// SignAccessToken signs a JWT access token and returns (jwt, jti, error).
func (km *KeyManager) SignAccessToken(issuer, clientID, subject string, groups []string, isAdmin bool) (string, string, error) {
	now := time.Now()
	jti := uuid.New().String()
	claimsJSON, err := json.Marshal(AccessTokenClaims{
		Sub:     subject,
		Iss:     issuer,
		Aud:     clientID,
		Exp:     now.Add(time.Hour).Unix(),
		Iat:     now.Unix(),
		JTI:     jti,
		Groups:  groups,
		IsAdmin: isAdmin,
	})
	if err != nil {
		return "", "", fmt.Errorf("failed to marshal claims: %w", err)
	}

	token, err := km.sign(jwtHeader{Alg: "RS256", Typ: "at+JWT", Kid: km.kid}, claimsJSON)
	if err != nil {
		return "", "", err
	}
	return token, jti, nil
}

// VerifyToken verifies an RS256 JWT and returns its access token claims.
func (km *KeyManager) VerifyToken(tokenString string) (*AccessTokenClaims, error) {
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid token format")
	}

	hash := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, fmt.Errorf("invalid signature encoding: %w", err)
	}
	if err := rsa.VerifyPKCS1v15(km.publicKey, stdcrypto.SHA256, hash[:], sig); err != nil {
		return nil, fmt.Errorf("invalid signature: %w", err)
	}

	claimsBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid claims encoding: %w", err)
	}

	var claims AccessTokenClaims
	if err := json.Unmarshal(claimsBytes, &claims); err != nil {
		return nil, fmt.Errorf("invalid claims: %w", err)
	}

	return &claims, nil
}
