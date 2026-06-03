package crypto

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/argon2"
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

// HashPassword hashes a password using Argon2id
func HashPassword(password string) (string, error) {
	// Generate a random salt
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	// Hash the password
	// Memory: 64MB, Iterations: 1, Parallelism: 4
	hash := argon2.IDKey([]byte(password), salt, 3, 64*1024, 4, 32)

	// Encode salt and hash to base64
	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	// Return in the format: $argon2id$v=19$m=65536,t=3,p=4$<salt>$<hash>
	return fmt.Sprintf("$argon2id$v=19$m=65536,t=3,p=4$%s$%s", b64Salt, b64Hash), nil
}

// VerifyPassword verifies a password against an argon2 hash
func VerifyPassword(password, encodedHash string) bool {
	// Expected format: $argon2id$v=19$m=65536,t=1,p=4$<salt>$<hash>
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 {
		return false
	}
	if parts[1] != "argon2id" || parts[2] != "v=19" {
		return false
	}

	memory, iterations, parallelism, err := parseArgon2idParams(parts[3])
	if err != nil {
		return false
	}

	// parts[1] is argon2id, parts[2] is v=19, parts[3] is parameters
	// parts[4] is salt, parts[5] is hash
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false
	}

	expectedHash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false
	}

	// Generate hash from provided password with the same salt
	computedHash := argon2.IDKey([]byte(password), salt, iterations, memory, parallelism, uint32(len(expectedHash)))

	// Constant-time comparison
	if len(computedHash) != len(expectedHash) {
		return false
	}

	return hmac.Equal(computedHash, expectedHash)
}

func parseArgon2idParams(encodedParams string) (memory uint32, iterations uint32, parallelism uint8, err error) {
	for _, param := range strings.Split(encodedParams, ",") {
		keyValue := strings.SplitN(param, "=", 2)
		if len(keyValue) != 2 {
			return 0, 0, 0, ErrInvalidToken
		}

		value, parseErr := strconv.ParseUint(keyValue[1], 10, 32)
		if parseErr != nil {
			return 0, 0, 0, parseErr
		}

		switch keyValue[0] {
		case "m":
			memory = uint32(value)
		case "t":
			iterations = uint32(value)
		case "p":
			if value > 255 {
				return 0, 0, 0, ErrInvalidToken
			}
			parallelism = uint8(value)
		}
	}

	if memory == 0 || iterations == 0 || parallelism == 0 {
		return 0, 0, 0, ErrInvalidToken
	}

	return memory, iterations, parallelism, nil
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

// computeHMAC computes HMAC-SHA256
func computeHMAC(data string, key []byte) string {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}
