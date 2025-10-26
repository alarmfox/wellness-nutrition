package crypto

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrInvalidSignature = errors.New("invalid signature")
	ErrExpiredToken     = errors.New("token has expired")
	ErrInvalidToken     = errors.New("invalid token format")
)

// SecretKey is the key used for signing tokens and cookies
// It should be loaded from an environment variable
var SecretKey []byte

// InitializeSecretKey sets the secret key from the provided string
func InitializeSecretKey(key string) error {
	if key == "" {
		return errors.New("secret key cannot be empty")
	}
	SecretKey = []byte(key)
	return nil
}

// SignToken creates a signed token with the given data
// Format: <data>.<signature>
func SignToken(data string) string {
	signature := computeHMAC(data, SecretKey)
	return fmt.Sprintf("%s.%s", data, signature)
}

// VerifyToken verifies a signed token and returns the original data
func VerifyToken(signedToken string) (string, error) {
	parts := strings.Split(signedToken, ".")
	if len(parts) != 2 {
		return "", ErrInvalidToken
	}

	data := parts[0]
	expectedSignature := parts[1]

	actualSignature := computeHMAC(data, SecretKey)

	if !hmac.Equal([]byte(expectedSignature), []byte(actualSignature)) {
		return "", ErrInvalidSignature
	}

	return data, nil
}

// SignedToken represents a token with expiration
type SignedToken struct {
	Data      string
	ExpiresAt int64 // Unix timestamp
}

// CreateTimedToken creates a signed token with expiration
// Format: <data>|<expiresAt>.<signature>
func CreateTimedToken(data string, expiresAt time.Time) string {
	payload := fmt.Sprintf("%s|%d", data, expiresAt.Unix())
	signature := computeHMAC(payload, SecretKey)
	return fmt.Sprintf("%s.%s", payload, signature)
}

// VerifyTimedToken verifies a timed token and returns the data if valid and not expired
func VerifyTimedToken(signedToken string) (string, error) {
	parts := strings.Split(signedToken, ".")
	if len(parts) != 2 {
		return "", ErrInvalidToken
	}

	payload := parts[0]
	expectedSignature := parts[1]

	actualSignature := computeHMAC(payload, SecretKey)

	if !hmac.Equal([]byte(expectedSignature), []byte(actualSignature)) {
		return "", ErrInvalidSignature
	}

	// Parse payload
	payloadParts := strings.Split(payload, "|")
	if len(payloadParts) != 2 {
		return "", ErrInvalidToken
	}

	data := payloadParts[0]
	var expiresAt int64
	_, err := fmt.Sscanf(payloadParts[1], "%d", &expiresAt)
	if err != nil {
		return "", ErrInvalidToken
	}

	// Check expiration
	if time.Now().Unix() > expiresAt {
		return "", ErrExpiredToken
	}

	return data, nil
}

// GenerateCSRFToken generates a new CSRF token
func GenerateCSRFToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	token := base64.URLEncoding.EncodeToString(b)
	// Sign the token
	return SignToken(token), nil
}

// VerifyCSRFToken verifies a CSRF token
func VerifyCSRFToken(token string) error {
	_, err := VerifyToken(token)
	return err
}

// computeHMAC computes HMAC-SHA256
func computeHMAC(data string, key []byte) string {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}
