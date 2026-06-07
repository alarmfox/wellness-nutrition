package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alarmfox/wellness-nutrition/app/crypto"
	"github.com/alarmfox/wellness-nutrition/app/models"
)

func TestPermissions(t *testing.T) {
	// Initialize crypto secret key for testing
	if err := crypto.InitializeSecretKey("test-secret-key-permissions"); err != nil {
		t.Fatalf("Failed to initialize secret key: %v", err)
	}

	sessionStore := newMockSessionStore()
	userRepo := newMockUserRepository()

	// Setup users
	adminUser := &models.User{ID: "admin-id", Role: models.RoleAdmin, Email: "admin@test.local"}
	normalUser := &models.User{ID: "user-id", Role: models.RoleUser, Email: "user@test.local"}
	userRepo.users[adminUser.ID] = adminUser
	userRepo.users[normalUser.ID] = normalUser

	// Setup sessions
	adminSessionID := "admin-token"
	adminSignedToken := crypto.CreateTimedToken(adminSessionID, time.Now().Add(time.Hour))
	sessionStore.sessions[adminSessionID] = &models.Session{Token: adminSessionID, UserID: adminUser.ID, ExpiresAt: time.Now().Add(time.Hour)}

	userSessionID := "user-token"
	userSignedToken := crypto.CreateTimedToken(userSessionID, time.Now().Add(time.Hour))
	sessionStore.sessions[userSessionID] = &models.Session{Token: userSessionID, UserID: normalUser.ID, ExpiresAt: time.Now().Add(time.Hour)}

	// Test handler
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	tests := []struct {
		name           string
		middleware     func(http.Handler) http.Handler
		token          string
		path           string
		expectedStatus int
	}{
		// Auth Middleware Tests (User Access)
		{
			name:           "Auth: Admin can access user area",
			middleware:     Auth(sessionStore, userRepo),
			token:          adminSignedToken,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Auth: Normal user can access user area",
			middleware:     Auth(sessionStore, userRepo),
			token:          userSignedToken,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Auth: Unauthenticated redirected to signin",
			middleware:     Auth(sessionStore, userRepo),
			token:          "",
			expectedStatus: http.StatusSeeOther,
		},

		// AdminAuth Middleware Tests (Admin Access)
		{
			name:           "AdminAuth: Admin can access admin area",
			middleware:     AdminAuth(sessionStore, userRepo),
			token:          adminSignedToken,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "AdminAuth: Normal user gets Forbidden",
			middleware:     AdminAuth(sessionStore, userRepo),
			token:          userSignedToken,
			path:           "/api/admin/users",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "AdminAuth: Unauthenticated gets Unauthorized",
			middleware:     AdminAuth(sessionStore, userRepo),
			token:          "",
			path:           "/api/admin/users",
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.path
			if path == "" {
				path = "/"
			}
			req := httptest.NewRequest("GET", path, nil)
			if tt.token != "" {
				req.AddCookie(&http.Cookie{Name: "session", Value: tt.token})
			}
			w := httptest.NewRecorder()

			tt.middleware(nextHandler).ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}
