package crypto

import (
	"encoding/base64"
	"fmt"
	"testing"
	"time"

	"golang.org/x/crypto/argon2"
)

func TestVerifyPassword(t *testing.T) {
	t.Run("current hash verifies", func(t *testing.T) {
		hash, err := HashPassword("correct-password")
		if err != nil {
			t.Fatalf("HashPassword() error = %v", err)
		}

		if !VerifyPassword("correct-password", hash) {
			t.Fatal("VerifyPassword() returned false for current hash")
		}
	})

	t.Run("stored t=3 hash verifies", func(t *testing.T) {
		hash := encodedArgon2idHash("correct-password", []byte("test-salt-123456"), 3, 64*1024, 4)

		if !VerifyPassword("correct-password", hash) {
			t.Fatal("VerifyPassword() returned false for t=3 hash")
		}
	})

	t.Run("invalid hash format fails", func(t *testing.T) {
		if VerifyPassword("correct-password", "not-an-argon2id-hash") {
			t.Fatal("VerifyPassword() returned true for invalid hash")
		}
	})

	t.Run("wrong password fails", func(t *testing.T) {
		hash, err := HashPassword("correct-password")
		if err != nil {
			t.Fatalf("HashPassword() error = %v", err)
		}

		if VerifyPassword("wrong-password", hash) {
			t.Fatal("VerifyPassword() returned true for wrong password")
		}
	})
}

func encodedArgon2idHash(password string, salt []byte, iterations uint32, memory uint32, parallelism uint8) string {
	hash := argon2.IDKey([]byte(password), salt, iterations, memory, parallelism, 32)
	return fmt.Sprintf(
		"$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		memory,
		iterations,
		parallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	)
}

func TestSignAndVerifyToken(t *testing.T) {
	// Initialize secret key for testing
	secretKey := "test-secret-key-for-testing-only"
	if err := InitializeSecretKey(secretKey); err != nil {
		t.Fatalf("Failed to initialize secret key: %v", err)
	}

	tests := []struct {
		name    string
		data    string
		wantErr bool
	}{
		{
			name:    "valid token",
			data:    "user123",
			wantErr: false,
		},
		{
			name:    "empty token",
			data:    "",
			wantErr: false,
		},
		{
			name:    "long token",
			data:    "very-long-token-with-lots-of-data-to-test-handling-of-larger-payloads",
			wantErr: false,
		},
		{
			name:    "token containing dots",
			data:    "payload.with.dots",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Sign the token
			signedToken := SignToken(tt.data)

			// Verify the token
			data, err := VerifyToken(signedToken)
			if (err != nil) != tt.wantErr {
				t.Errorf("VerifyToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && data != tt.data {
				t.Errorf("VerifyToken() got = %v, want %v", data, tt.data)
			}
		})
	}
}

func TestVerifyTokenWithInvalidSignature(t *testing.T) {
	// Initialize secret key
	secretKey := "test-secret-key-for-testing-only"
	if err := InitializeSecretKey(secretKey); err != nil {
		t.Fatalf("Failed to initialize secret key: %v", err)
	}

	// Sign a token
	signedToken := SignToken("user123")

	// Modify the signature
	parts := signedToken[:len(signedToken)-5] + "xxxxx"
	_, err := VerifyToken(parts)
	if err != ErrInvalidSignature {
		t.Errorf("Expected ErrInvalidSignature for modified signature, got %v", err)
	}

	// Test with completely invalid format (no dot separator)
	_, err = VerifyToken("invalid-token-without-dot")
	if err != ErrInvalidToken {
		t.Errorf("Expected ErrInvalidToken for token without separator, got %v", err)
	}
}

func TestCreateAndVerifyTimedToken(t *testing.T) {
	// Initialize secret key
	secretKey := "test-secret-key-for-testing-only"
	if err := InitializeSecretKey(secretKey); err != nil {
		t.Fatalf("Failed to initialize secret key: %v", err)
	}

	tests := []struct {
		name      string
		data      string
		expiresAt time.Time
		wantErr   error
	}{
		{
			name:      "valid token not expired",
			data:      "user123",
			expiresAt: time.Now().Add(1 * time.Hour),
			wantErr:   nil,
		},
		{
			name:      "valid token with separators not expired",
			data:      "user.with.dots|and-pipes",
			expiresAt: time.Now().Add(1 * time.Hour),
			wantErr:   nil,
		},
		{
			name:      "token expired",
			data:      "user123",
			expiresAt: time.Now().Add(-1 * time.Hour),
			wantErr:   ErrExpiredToken,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create timed token
			signedToken := CreateTimedToken(tt.data, tt.expiresAt)

			// Verify the token
			data, err := VerifyTimedToken(signedToken)

			if err != tt.wantErr {
				t.Errorf("VerifyTimedToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr == nil && data != tt.data {
				t.Errorf("VerifyTimedToken() got = %v, want %v", data, tt.data)
			}
		})
	}
}

func TestVerifyTimedTokenWithTampering(t *testing.T) {
	// Initialize secret key
	secretKey := "test-secret-key-for-testing-only"
	if err := InitializeSecretKey(secretKey); err != nil {
		t.Fatalf("Failed to initialize secret key: %v", err)
	}

	// Create a valid token
	expiresAt := time.Now().Add(1 * time.Hour)
	signedToken := CreateTimedToken("user123", expiresAt)

	// Tamper with the data part
	tamperedToken := "user456|" + signedToken[len("user123|"):]

	// Verify should fail
	_, err := VerifyTimedToken(tamperedToken)
	if err != ErrInvalidSignature {
		t.Errorf("Expected ErrInvalidSignature for tampered data, got %v", err)
	}
}

func TestGenerateCSRFTokenUniqueness(t *testing.T) {
	// Initialize secret key
	secretKey := "test-secret-key-for-testing-only"
	if err := InitializeSecretKey(secretKey); err != nil {
		t.Fatalf("Failed to initialize secret key: %v", err)
	}

	// Generate multiple signed tokens and ensure they're unique
	// Note: This tests the signing mechanism, not CSRF token generation
	// which is in the middleware package
	tokens := make(map[string]bool)
	for i := 0; i < 100; i++ {
		token := SignToken(time.Now().String())
		if tokens[token] {
			t.Errorf("Generated duplicate token: %s", token)
		}
		tokens[token] = true
	}
}

func TestSecretKeyInitialization(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		wantErr bool
	}{
		{
			name:    "valid key",
			key:     "valid-secret-key",
			wantErr: false,
		},
		{
			name:    "empty key",
			key:     "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := InitializeSecretKey(tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("InitializeSecretKey() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
