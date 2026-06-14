package crypto

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/big"
	"time"
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
		kid:        "default",
	}, nil
}

type JWKSResponse struct {
	Keys []JWK `json:"keys"`
}

type JWK struct {
	Kty string `json:"kty"`
	Use string `json:"use"`
	Kid string `json:"kid"`
	N   string `json:"n"`
	E   string `json:"e"`
}

func (km *KeyManager) JWKS() JWKSResponse {
	n := km.publicKey.N
	e := km.publicKey.E

	return JWKSResponse{
		Keys: []JWK{
			{
				Kty: "RSA",
				Use: "sig",
				Kid: km.kid,
				N:   base64.RawURLEncoding.EncodeToString(n.Bytes()),
				E:   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(e)).Bytes()),
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
	Email         string   `json:"email,omitempty"`
	EmailVerified bool     `json:"email_verified,omitempty"`
	Name          string   `json:"name,omitempty"`
	Groups        []string `json:"groups,omitempty"`
}

type jwtHeader struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
	Kid string `json:"kid"`
}

func (km *KeyManager) SignIDToken(issuer, clientID, subject, email, name string, groups []string) (string, error) {
	now := time.Now()
	exp := now.Add(1 * time.Hour)

	claims := IDTokenClaims{
		Sub:           subject,
		Iss:           issuer,
		Aud:           clientID,
		Exp:           exp.Unix(),
		Iat:           now.Unix(),
		Email:         email,
		EmailVerified: true,
		Name:          name,
		Groups:        groups,
	}

	header := jwtHeader{
		Alg: "RS256",
		Typ: "JWT",
		Kid: km.kid,
	}

	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", fmt.Errorf("failed to marshal header: %w", err)
	}

	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("failed to marshal claims: %w", err)
	}

	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)
	claimsB64 := base64.RawURLEncoding.EncodeToString(claimsJSON)
	message := headerB64 + "." + claimsB64

	hash := sha256.Sum256([]byte(message))
	signature, err := rsa.SignPKCS1v15(rand.Reader, km.privateKey, crypto.SHA256, hash[:])
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	signatureB64 := base64.RawURLEncoding.EncodeToString(signature)
	return message + "." + signatureB64, nil
}
