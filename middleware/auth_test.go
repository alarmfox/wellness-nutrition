package middleware

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alarmfox/wellness-nutrition/app/crypto"
	"github.com/alarmfox/wellness-nutrition/app/models"
)

// Mock SessionStore for testing - implements SessionStoreInterface
type mockSessionStore struct {
	sessions       map[string]*models.Session
	extendedTokens map[string]string // old token -> new token
}

func newMockSessionStore() *mockSessionStore {
	return &mockSessionStore{
		sessions:       make(map[string]*models.Session),
		extendedTokens: make(map[string]string),
	}
}

func (m *mockSessionStore) GetSession(signedToken string) (*models.Session, error) {
	sessionID, err := crypto.VerifyTimedToken(signedToken)
	if err != nil {
		return nil, err
	}

	session, ok := m.sessions[sessionID]
	if !ok {
		return nil, sql.ErrNoRows
	}

	return session, nil
}

func (m *mockSessionStore) ExtendSession(signedToken string, newExpiresAt time.Time) (string, error) {
	sessionID, err := crypto.VerifyTimedToken(signedToken)
	if err != nil {
		return "", err
	}

	session, ok := m.sessions[sessionID]
	if !ok {
		return "", sql.ErrNoRows
	}

	session.ExpiresAt = newExpiresAt
	newToken := crypto.CreateTimedToken(sessionID, newExpiresAt)
	m.extendedTokens[signedToken] = newToken

	return newToken, nil
}

// Mock UserRepository for testing - implements UserRepositoryInterface
type mockUserRepository struct {
	users map[string]*models.User
}

func newMockUserRepository() *mockUserRepository {
	return &mockUserRepository{
		users: make(map[string]*models.User),
	}
}

func (m *mockUserRepository) GetByID(id string) (*models.User, error) {
	user, ok := m.users[id]
	if !ok {
		return nil, sql.ErrNoRows
	}
	return user, nil
}

func TestSessionExtension(t *testing.T) {
	// Initialize crypto secret key for testing
	if err := crypto.InitializeSecretKey("test-secret-key"); err != nil {
		t.Fatalf("Failed to initialize secret key: %v", err)
	}

	sessionStore := newMockSessionStore()
	userRepo := newMockUserRepository()

	// Create a test user
	user := &models.User{
		ID:        "user-1",
		FirstName: "Test",
		LastName:  "User",
		Email:     "test@example.com",
		Role:      models.RoleUser,
	}
	userRepo.users[user.ID] = user

	t.Run("session is extended when near expiration", func(t *testing.T) {
		// Create a session that expires in 5 days (less than the 7-day threshold)
		sessionID := "test-session-near-expiry"
		expiresAt := time.Now().Add(5 * 24 * time.Hour)
		signedToken := crypto.CreateTimedToken(sessionID, expiresAt)

		sessionStore.sessions[sessionID] = &models.Session{
			Token:     sessionID,
			UserID:    user.ID,
			ExpiresAt: expiresAt,
		}

		// Create a test handler
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		})

		// Wrap with Auth middleware
		authHandler := Auth(sessionStore, userRepo)(handler)

		// Create request with the session cookie
		req := httptest.NewRequest("GET", "/test", nil)
		req.AddCookie(&http.Cookie{
			Name:  "session",
			Value: signedToken,
		})
		w := httptest.NewRecorder()

		// Execute request
		authHandler.ServeHTTP(w, req)

		// Check response
		if w.Code != http.StatusOK {
			t.Errorf("Expected status OK, got %d", w.Code)
		}

		// Check if a new session cookie was set
		cookies := w.Result().Cookies()
		var sessionCookie *http.Cookie
		for _, c := range cookies {
			if c.Name == "session" {
				sessionCookie = c
				break
			}
		}

		if sessionCookie == nil {
			t.Error("Expected new session cookie to be set, but none was found")
			return
		}

		// Verify the new token is different from the old one
		if sessionCookie.Value == signedToken {
			t.Error("Expected new token to be different from old token")
		}

		// Verify the new token is valid
		newSessionID, err := crypto.VerifyTimedToken(sessionCookie.Value)
		if err != nil {
			t.Errorf("New token is invalid: %v", err)
		}

		if newSessionID != sessionID {
			t.Errorf("Expected session ID %s, got %s", sessionID, newSessionID)
		}
	})

	t.Run("session is not extended when far from expiration", func(t *testing.T) {
		// Create a session that expires in 20 days (more than the 7-day threshold)
		sessionID := "test-session-far-expiry"
		expiresAt := time.Now().Add(20 * 24 * time.Hour)
		signedToken := crypto.CreateTimedToken(sessionID, expiresAt)

		sessionStore.sessions[sessionID] = &models.Session{
			Token:     sessionID,
			UserID:    user.ID,
			ExpiresAt: expiresAt,
		}

		// Create a test handler
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		})

		// Wrap with Auth middleware
		authHandler := Auth(sessionStore, userRepo)(handler)

		// Create request with the session cookie
		req := httptest.NewRequest("GET", "/test", nil)
		req.AddCookie(&http.Cookie{
			Name:  "session",
			Value: signedToken,
		})
		w := httptest.NewRecorder()

		// Execute request
		authHandler.ServeHTTP(w, req)

		// Check response
		if w.Code != http.StatusOK {
			t.Errorf("Expected status OK, got %d", w.Code)
		}

		// Check that no new session cookie was set
		cookies := w.Result().Cookies()
		var sessionCookie *http.Cookie
		for _, c := range cookies {
			if c.Name == "session" {
				sessionCookie = c
				break
			}
		}

		// If a cookie was set, it should be the same token (no extension)
		if sessionCookie != nil && sessionCookie.Value != signedToken {
			t.Error("Session should not have been extended")
		}
	})

	t.Run("admin session is extended when near expiration", func(t *testing.T) {
		// Create an admin user
		adminUser := &models.User{
			ID:        "admin-1",
			FirstName: "Admin",
			LastName:  "User",
			Email:     "admin@example.com",
			Role:      models.RoleAdmin,
		}
		userRepo.users[adminUser.ID] = adminUser

		// Create a session that expires in 5 days
		sessionID := "admin-session-near-expiry"
		expiresAt := time.Now().Add(5 * 24 * time.Hour)
		signedToken := crypto.CreateTimedToken(sessionID, expiresAt)

		sessionStore.sessions[sessionID] = &models.Session{
			Token:     sessionID,
			UserID:    adminUser.ID,
			ExpiresAt: expiresAt,
		}

		// Create a test handler
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		})

		// Wrap with AdminAuth middleware
		adminAuthHandler := AdminAuth(sessionStore, userRepo)(handler)

		// Create request with the session cookie
		req := httptest.NewRequest("GET", "/admin", nil)
		req.AddCookie(&http.Cookie{
			Name:  "session",
			Value: signedToken,
		})
		w := httptest.NewRecorder()

		// Execute request
		adminAuthHandler.ServeHTTP(w, req)

		// Check response
		if w.Code != http.StatusOK {
			t.Errorf("Expected status OK, got %d", w.Code)
		}

		// Check if a new session cookie was set
		cookies := w.Result().Cookies()
		var sessionCookie *http.Cookie
		for _, c := range cookies {
			if c.Name == "session" {
				sessionCookie = c
				break
			}
		}

		if sessionCookie == nil {
			t.Error("Expected new session cookie to be set, but none was found")
			return
		}

		// Verify the new token is different from the old one
		if sessionCookie.Value == signedToken {
			t.Error("Expected new token to be different from old token")
		}
	})

	t.Run("user context is still set correctly after extension", func(t *testing.T) {
		// Create a session that expires in 5 days
		sessionID := "test-session-context"
		expiresAt := time.Now().Add(5 * 24 * time.Hour)
		signedToken := crypto.CreateTimedToken(sessionID, expiresAt)

		sessionStore.sessions[sessionID] = &models.Session{
			Token:     sessionID,
			UserID:    user.ID,
			ExpiresAt: expiresAt,
		}

		// Create a test handler that checks the user context
		var ctxUser *models.User
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctxUser = GetUserFromContext(r.Context())
			w.WriteHeader(http.StatusOK)
		})

		// Wrap with Auth middleware
		authHandler := Auth(sessionStore, userRepo)(handler)

		// Create request
		req := httptest.NewRequest("GET", "/test", nil)
		req.AddCookie(&http.Cookie{
			Name:  "session",
			Value: signedToken,
		})
		w := httptest.NewRecorder()

		// Execute request
		authHandler.ServeHTTP(w, req)

		// Verify user context
		if ctxUser == nil {
			t.Error("User context is nil")
			return
		}

		if ctxUser.ID != user.ID {
			t.Errorf("Expected user ID %s, got %s", user.ID, ctxUser.ID)
		}
	})
}

func TestGetUserFromContext(t *testing.T) {
	user := &models.User{
		ID:        "user-1",
		FirstName: "Test",
		LastName:  "User",
		Email:     "test@example.com",
	}

	ctx := context.WithValue(context.Background(), UserContextKey, user)

	retrievedUser := GetUserFromContext(ctx)
	if retrievedUser == nil {
		t.Error("Expected user from context, got nil")
		return
	}

	if retrievedUser.ID != user.ID {
		t.Errorf("Expected user ID %s, got %s", user.ID, retrievedUser.ID)
	}
}

func TestGetUserFromContextNil(t *testing.T) {
	ctx := context.Background()

	retrievedUser := GetUserFromContext(ctx)
	if retrievedUser != nil {
		t.Error("Expected nil user from empty context")
	}
}
