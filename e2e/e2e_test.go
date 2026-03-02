// +build e2e

package e2e

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alarmfox/wellness-nutrition/app/crypto"
	"github.com/alarmfox/wellness-nutrition/app/middleware"
	"github.com/alarmfox/wellness-nutrition/app/models"
	"github.com/alarmfox/wellness-nutrition/app/testutil"
	"github.com/google/uuid"
)

// TestLoginEndpointExists tests that the login endpoint is accessible
func TestLoginEndpointExists(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e test in short mode")
	}

	// Initialize crypto
	if err := crypto.InitializeSecretKey("test-secret-key"); err != nil {
		t.Fatalf("Failed to initialize secret key: %v", err)
	}

	// Create a simple handler that mimics the login page
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Login Page"))
	})

	t.Run("Get login page with CSRF protection", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/login", nil)
		w := httptest.NewRecorder()

		// Wrap with CSRF middleware
		csrfHandler := middleware.CSRF(handler)
		csrfHandler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		// Check that CSRF cookie is set
		cookies := w.Result().Cookies()
		found := false
		for _, c := range cookies {
			if c.Name == middleware.CSRFCookieName {
				found = true
				break
			}
		}
		if !found {
			t.Error("CSRF cookie not set")
		}
	})
}

// TestAuthenticationMiddleware tests the authentication middleware
func TestAuthenticationMiddleware(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e test in short mode")
	}

	// Initialize crypto
	if err := crypto.InitializeSecretKey("test-secret-key"); err != nil {
		t.Fatalf("Failed to initialize secret key: %v", err)
	}

	// Setup test database
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	testutil.CreateTestSchema(t, db)
	defer testutil.DropTestSchema(t, db)

	// Create repositories
	userRepo := models.NewUserRepository(db)
	sessionStore := models.NewSessionStore(db)

	// Create test user
	user := &models.User{
		ID:                uuid.New().String(),
		FirstName:         "Test",
		LastName:          "User",
		Email:             "testuser@example.com",
		Role:              models.RoleUser,
		SubType:           models.SubTypeSingle,
		ExpiresAt:         time.Now().Add(30 * 24 * time.Hour),
		RemainingAccesses: 10,
	}

	err := userRepo.Create(user)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Create session
	signedToken, err := sessionStore.CreateSession(user.ID)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Protected handler
	protectedHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := middleware.GetUserFromContext(r.Context())
		if user == nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Protected Resource"))
	})

	t.Run("Authenticated request succeeds", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/protected", nil)
		req.AddCookie(&http.Cookie{
			Name:  "session",
			Value: signedToken,
		})
		w := httptest.NewRecorder()

		handler := middleware.Auth(sessionStore, userRepo)(protectedHandler)
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("Unauthenticated request is rejected", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/protected", nil)
		w := httptest.NewRecorder()

		handler := middleware.Auth(sessionStore, userRepo)(protectedHandler)
		handler.ServeHTTP(w, req)

		if w.Code == http.StatusOK {
			t.Errorf("Expected non-OK status, got %d", w.Code)
		}
	})
}

// TestAdminAuthenticationMiddleware tests the admin authentication middleware
func TestAdminAuthenticationMiddleware(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e test in short mode")
	}

	// Initialize crypto
	if err := crypto.InitializeSecretKey("test-secret-key"); err != nil {
		t.Fatalf("Failed to initialize secret key: %v", err)
	}

	// Setup test database
	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	testutil.CreateTestSchema(t, db)
	defer testutil.DropTestSchema(t, db)

	// Create repositories
	userRepo := models.NewUserRepository(db)
	sessionStore := models.NewSessionStore(db)

	// Create regular user
	regularUser := &models.User{
		ID:                uuid.New().String(),
		FirstName:         "Regular",
		LastName:          "User",
		Email:             "user@example.com",
		Role:              models.RoleUser,
		SubType:           models.SubTypeSingle,
		ExpiresAt:         time.Now().Add(30 * 24 * time.Hour),
		RemainingAccesses: 10,
	}
	err := userRepo.Create(regularUser)
	if err != nil {
		t.Fatalf("Failed to create regular user: %v", err)
	}

	// Create admin user
	adminUser := &models.User{
		ID:                uuid.New().String(),
		FirstName:         "Admin",
		LastName:          "User",
		Email:             "admin@example.com",
		Role:              models.RoleAdmin,
		SubType:           models.SubTypeSingle,
		ExpiresAt:         time.Now().Add(30 * 24 * time.Hour),
		RemainingAccesses: 0,
	}
	err = userRepo.Create(adminUser)
	if err != nil {
		t.Fatalf("Failed to create admin user: %v", err)
	}

	// Create sessions
	regularToken, err := sessionStore.CreateSession(regularUser.ID)
	if err != nil {
		t.Fatalf("Failed to create regular session: %v", err)
	}

	adminToken, err := sessionStore.CreateSession(adminUser.ID)
	if err != nil {
		t.Fatalf("Failed to create admin session: %v", err)
	}

	// Admin-only handler
	adminHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Admin Resource"))
	})

	t.Run("Admin user can access admin endpoint", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/admin", nil)
		req.AddCookie(&http.Cookie{
			Name:  "session",
			Value: adminToken,
		})
		w := httptest.NewRecorder()

		handler := middleware.AdminAuth(sessionStore, userRepo)(adminHandler)
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 for admin, got %d", w.Code)
		}
	})

	t.Run("Regular user cannot access admin endpoint", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/admin", nil)
		req.AddCookie(&http.Cookie{
			Name:  "session",
			Value: regularToken,
		})
		w := httptest.NewRecorder()

		handler := middleware.AdminAuth(sessionStore, userRepo)(adminHandler)
		handler.ServeHTTP(w, req)

		if w.Code == http.StatusOK {
			t.Errorf("Expected non-OK status for regular user, got %d", w.Code)
		}
	})
}

// TestCSRFProtection tests CSRF protection on unsafe methods
func TestCSRFProtection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping e2e test in short mode")
	}

	// Initialize crypto
	if err := crypto.InitializeSecretKey("test-secret-key"); err != nil {
		t.Fatalf("Failed to initialize secret key: %v", err)
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Success"))
	})

	t.Run("POST without CSRF token is rejected", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/action", nil)
		w := httptest.NewRecorder()

		csrfHandler := middleware.CSRF(handler)
		csrfHandler.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("Expected status 403, got %d", w.Code)
		}
	})

	t.Run("GET request succeeds and sets CSRF token", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/action", nil)
		w := httptest.NewRecorder()

		csrfHandler := middleware.CSRF(handler)
		csrfHandler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		// Check CSRF cookie
		cookies := w.Result().Cookies()
		var csrfCookie *http.Cookie
		for _, c := range cookies {
			if c.Name == middleware.CSRFCookieName {
				csrfCookie = c
				break
			}
		}

		if csrfCookie == nil {
			t.Error("CSRF cookie not set")
		}
	})

	t.Run("POST with valid CSRF token succeeds", func(t *testing.T) {
		// First, get a CSRF token
		getReq := httptest.NewRequest("GET", "/api/action", nil)
		getW := httptest.NewRecorder()

		csrfHandler := middleware.CSRF(handler)
		csrfHandler.ServeHTTP(getW, getReq)

		// Extract CSRF token
		var csrfToken string
		cookies := getW.Result().Cookies()
		for _, c := range cookies {
			if c.Name == middleware.CSRFCookieName {
				csrfToken = c.Value
				break
			}
		}

		if csrfToken == "" {
			t.Fatal("CSRF token not found")
		}

		// Now make POST request with CSRF token
		postReq := httptest.NewRequest("POST", "/api/action", nil)
		postReq.Header.Set(middleware.CSRFHeaderName, csrfToken)
		postReq.AddCookie(&http.Cookie{
			Name:  middleware.CSRFCookieName,
			Value: csrfToken,
		})
		postW := httptest.NewRecorder()

		csrfHandler.ServeHTTP(postW, postReq)

		if postW.Code != http.StatusOK {
			t.Errorf("Expected status 200 with valid CSRF token, got %d", postW.Code)
		}
	})
}

// Mock session store for tests without database
type mockSessionStore struct{}

func (m *mockSessionStore) GetSession(signedToken string) (*models.Session, error) {
	return nil, sql.ErrNoRows
}

func (m *mockSessionStore) ExtendSession(signedToken string, newExpiresAt time.Time) (string, error) {
	return "", sql.ErrNoRows
}
