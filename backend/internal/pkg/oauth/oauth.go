// Package oauth provides shared OAuth utilities.
package oauth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"strings"
)

// GenerateState generates a cryptographically random state parameter for OAuth flows.
func GenerateState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func GenerateRandomBytes(size int) ([]byte, error) {
	b := make([]byte, size)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}
	return b, nil
}

// GenerateCodeVerifier returns an RFC 7636 PKCE verifier.
func GenerateCodeVerifier() (string, error) {
	b, err := GenerateRandomBytes(32)
	if err != nil {
		return "", err
	}
	return base64URLEncode(b), nil
}

// GenerateCodeChallenge returns an S256 PKCE challenge for a verifier.
func GenerateCodeChallenge(verifier string) string {
	hash := sha256.Sum256([]byte(verifier))
	return base64URLEncode(hash[:])
}

func base64URLEncode(data []byte) string {
	return strings.TrimRight(base64.URLEncoding.EncodeToString(data), "=")
}
