package registryauth

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// TokenGenerator handles JWT token generation for Docker registry authentication
type TokenGenerator struct {
	privateKey *rsa.PrivateKey
	issuer     string
	expiration time.Duration
}

// AccessEntry represents a Docker registry access grant
type AccessEntry struct {
	Type    string   `json:"type"`
	Name    string   `json:"name"`
	Actions []string `json:"actions"`
}

// TokenClaims represents the JWT claims for Docker registry
type TokenClaims struct {
	jwt.RegisteredClaims
	Access []AccessEntry `json:"access"`
}

// NewTokenGenerator creates a new token generator by loading the RSA private key
func NewTokenGenerator(privateKeyPath, issuer string, expirationSec int) (*TokenGenerator, error) {
	// Read private key file
	keyData, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key: %w", err)
	}

	// Parse PEM format
	block, _ := pem.Decode(keyData)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block from private key")
	}

	// Parse RSA private key
	privateKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		// Try PKCS1 format as fallback
		privateKey, err = x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key: %w", err)
		}
	}

	rsaKey, ok := privateKey.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("private key is not RSA format")
	}

	return &TokenGenerator{
		privateKey: rsaKey,
		issuer:     issuer,
		expiration: time.Duration(expirationSec) * time.Second,
	}, nil
}

// GenerateToken creates a JWT token with the specified access grants
func (tg *TokenGenerator) GenerateToken(subject, audience string, access []AccessEntry) (string, time.Time, error) {
	now := time.Now()
	expiresAt := now.Add(tg.expiration)

	claims := TokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    tg.issuer,
			Subject:   subject,
			Audience:  jwt.ClaimStrings{audience},
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        uuid.New().String(),
		},
		Access: access,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	// Set the Key ID (kid) header to match the JWKS key ID
	token.Header["kid"] = "registry-auth-jwt-signer"

	tokenString, err := token.SignedString(tg.privateKey)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, expiresAt, nil
}
