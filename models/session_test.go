package models

import (
	"testing"
	"time"

	"github.com/alarmfox/wellness-nutrition/app/crypto"
)

func TestExtendSessionMethodExists(t *testing.T) {
	// Initialize crypto secret key for testing
	if err := crypto.InitializeSecretKey("test-secret-key-for-session"); err != nil {
		t.Fatalf("Failed to initialize secret key: %v", err)
	}

	// This test just verifies that the ExtendSession method exists and has the correct signature
	// Real testing would require a database connection

	t.Run("method signature is correct", func(t *testing.T) {
		// Create a nil session store (won't use it)
		var store *SessionStore

		// Verify method exists by creating a function pointer
		var extendFunc func(string, time.Time) (string, error)
		if store != nil {
			extendFunc = store.ExtendSession
		}

		if extendFunc != nil {
			t.Error("Store should be nil for this test")
		}

		// If we got here without compilation errors, the method signature is correct
	})

	t.Run("token format validation", func(t *testing.T) {
		// Test that we can create and verify tokens with different expiration times
		sessionID := "test-session-id"

		// Create token with initial expiration
		oldExpiresAt := time.Now().Add(5 * 24 * time.Hour)
		oldToken := crypto.CreateTimedToken(sessionID, oldExpiresAt)

		// Verify old token
		verifiedOldID, err := crypto.VerifyTimedToken(oldToken)
		if err != nil {
			t.Fatalf("Old token verification failed: %v", err)
		}
		if verifiedOldID != sessionID {
			t.Errorf("Expected session ID %s, got %s", sessionID, verifiedOldID)
		}

		// Create token with new expiration
		newExpiresAt := time.Now().Add(30 * 24 * time.Hour)
		newToken := crypto.CreateTimedToken(sessionID, newExpiresAt)

		// Verify new token
		verifiedNewID, err := crypto.VerifyTimedToken(newToken)
		if err != nil {
			t.Fatalf("New token verification failed: %v", err)
		}
		if verifiedNewID != sessionID {
			t.Errorf("Expected session ID %s, got %s", sessionID, verifiedNewID)
		}

		// Tokens should be different
		if oldToken == newToken {
			t.Error("Tokens with different expirations should be different")
		}

		// But they should both contain the same session ID
		if verifiedOldID != verifiedNewID {
			t.Error("Both tokens should contain the same session ID")
		}
	})
}
