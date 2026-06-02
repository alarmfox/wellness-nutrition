package middleware

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/alarmfox/wellness-nutrition/app/models"
)

// SessionStoreInterface defines the interface for session management
type SessionStoreInterface interface {
	GetSession(token string) (*models.Session, error)
	ExtendSession(signedToken string, newExpiresAt time.Time) (string, error)
}

// UserRepositoryInterface defines the interface for user management
type UserRepositoryInterface interface {
	GetByID(id string) (*models.User, error)
}

type contextKey string

const (
	UserContextKey contextKey = "user"
	// SessionExtensionThreshold is the duration before expiration when we extend sessions
	// If a session has less than this time remaining, it will be extended
	SessionExtensionThreshold = 7 * 24 * time.Hour // 7 days
	// SessionDuration is how long a session lasts
	SessionDuration = 30 * 24 * time.Hour // 30 days
)

// Auth middleware checks if user is authenticated
func Auth(sessionStore SessionStoreInterface, userRepo UserRepositoryInterface) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("session")
			if err != nil {
				http.Redirect(w, r, "/signin", http.StatusSeeOther)
				return
			}

			session, err := sessionStore.GetSession(cookie.Value)
			if err != nil {
				http.Redirect(w, r, "/signin", http.StatusSeeOther)
				return
			}

			user, err := userRepo.GetByID(session.UserID)
			if err != nil {
				http.Redirect(w, r, "/signin", http.StatusSeeOther)
				return
			}

			// Extend session if needed (transparent to the rest of the application)
			extendSessionIfNeeded(w, sessionStore, session, cookie.Value)

			ctx := context.WithValue(r.Context(), UserContextKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// AdminAuth middleware checks if user is authenticated and is an admin
func AdminAuth(sessionStore SessionStoreInterface, userRepo UserRepositoryInterface) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("session")
			if err != nil {
				sendJSON(w, http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
				return
			}

			session, err := sessionStore.GetSession(cookie.Value)
			if err != nil {
				sendJSON(w, http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
				return
			}

			user, err := userRepo.GetByID(session.UserID)
			if err != nil {
				sendJSON(w, http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
				return
			}

			if user.Role != models.RoleAdmin {
				sendJSON(w, http.StatusForbidden, map[string]string{"error": "Forbidden - Admin access required"})
				return
			}

			// Extend session if needed (transparent to the rest of the application)
			extendSessionIfNeeded(w, sessionStore, session, cookie.Value)

			ctx := context.WithValue(r.Context(), UserContextKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUserFromContext retrieves user from context
func GetUserFromContext(ctx context.Context) *models.User {
	user, ok := ctx.Value(UserContextKey).(*models.User)
	if !ok {
		return nil
	}
	return user
}

// extendSessionIfNeeded checks if a session is about to expire and extends it
// Returns the new token if extended, or the original token if no extension was needed
func extendSessionIfNeeded(w http.ResponseWriter, sessionStore SessionStoreInterface, session *models.Session, currentToken string) string {
	timeUntilExpiration := time.Until(session.ExpiresAt)

	// If session expires in less than the threshold, extend it
	if timeUntilExpiration < SessionExtensionThreshold {
		newExpiresAt := time.Now().Add(SessionDuration)
		newToken, err := sessionStore.ExtendSession(currentToken, newExpiresAt)
		if err != nil {
			// If extension fails, log the error but don't interrupt the request
			log.Printf("Failed to extend session: %v", err)
			return currentToken
		}

		// Set the new token as a cookie
		isProd := os.Getenv("ENVIRONMENT") == "production"
		http.SetCookie(w, &http.Cookie{
			Name:     "session",
			Value:    newToken,
			Path:     "/",
			HttpOnly: true,
			Secure:   isProd,
			SameSite: http.SameSiteLaxMode,
			Expires:  newExpiresAt,
		})

		return newToken
	}

	return currentToken
}

func sendJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Error encoding JSON: %v", err)
	}
}
