package middleware

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"

	"github.com/alarmfox/wellness-nutrition/app/models"
	"golang.org/x/crypto/argon2"
)

type contextKey string

const (
	UserContextKey contextKey = "user"
)

// Hash password using argon2
func HashPassword(password string) string {
	salt := make([]byte, 16)
	rand.Read(salt)

	hash := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)

	encoded := "$argon2id$v=19$m=65536,t=1,p=4$" +
		base64.RawStdEncoding.EncodeToString(salt) + "$" +
		base64.RawStdEncoding.EncodeToString(hash)

	return encoded
}

// VerifyPassword verifies a password against an argon2 hash
func VerifyPassword(password, encodedHash string) bool {
	// Parse the encoded hash
	// Expected format: $argon2id$v=19$m=65536,t=1,p=4$<salt>$<hash>
	parts := []byte(encodedHash)

	// Find salt and hash parts
	dollarCount := 0
	saltStart := 0
	hashStart := 0

	for i, b := range parts {
		if b == '$' {
			dollarCount++
			if dollarCount == 4 {
				saltStart = i + 1
			} else if dollarCount == 5 {
				hashStart = i + 1
				break
			}
		}
	}

	if hashStart == 0 {
		// Invalid format, fallback to direct comparison for backward compatibility
		return password == encodedHash
	}

	// Extract salt and hash
	saltStr := string(parts[saltStart : hashStart-1])
	hashStr := string(parts[hashStart:])

	salt, err := base64.RawStdEncoding.DecodeString(saltStr)
	if err != nil {
		return false
	}

	expectedHash, err := base64.RawStdEncoding.DecodeString(hashStr)
	if err != nil {
		return false
	}

	// Generate hash from provided password with the same salt
	computedHash := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)

	// Compare hashes
	if len(computedHash) != len(expectedHash) {
		return false
	}

	for i := range computedHash {
		if computedHash[i] != expectedHash[i] {
			return false
		}
	}

	return true
}

// Auth middleware checks if user is authenticated
func Auth(sessionStore *models.SessionStore, userRepo *models.UserRepository) func(http.Handler) http.Handler {
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

			ctx := context.WithValue(r.Context(), UserContextKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// AdminAuth middleware checks if user is authenticated and is an admin
func AdminAuth(sessionStore *models.SessionStore, userRepo *models.UserRepository) func(http.Handler) http.Handler {
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

func sendJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Error encoding JSON: %v", err)
	}
}
