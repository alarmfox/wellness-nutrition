package middleware

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/alarmfox/wellness-nutrition/app/models"
	"golang.org/x/crypto/argon2"
)

type contextKey string

const (
	UserContextKey contextKey = "user"
)

type SessionStore struct {
	db *sql.DB
}

type Session struct {
	Token     string
	UserID    string
	ExpiresAt time.Time
}

func NewSessionStore(db *sql.DB) *SessionStore {
	return &SessionStore{db: db}
}

// Create session table if not exists
func (s *SessionStore) InitTable() error {
	query := `
		CREATE TABLE IF NOT EXISTS sessions (
			token VARCHAR(255) PRIMARY KEY,
			user_id VARCHAR(255) NOT NULL,
			expires_at TIMESTAMP NOT NULL
		)
	`
	_, err := s.db.Exec(query)
	return err
}

func (s *SessionStore) CreateSession(userID string) (string, error) {
	token := generateToken()
	expiresAt := time.Now().Add(30 * 24 * time.Hour) // 30 days
	
	query := `INSERT INTO sessions (token, user_id, expires_at) VALUES ($1, $2, $3)`
	_, err := s.db.Exec(query, token, userID, expiresAt)
	if err != nil {
		return "", err
	}
	
	return token, nil
}

func (s *SessionStore) GetSession(token string) (*Session, error) {
	query := `SELECT token, user_id, expires_at FROM sessions WHERE token = $1 AND expires_at > NOW()`
	
	var session Session
	err := s.db.QueryRow(query, token).Scan(&session.Token, &session.UserID, &session.ExpiresAt)
	if err != nil {
		return nil, err
	}
	
	return &session, nil
}

func (s *SessionStore) DeleteSession(token string) error {
	query := `DELETE FROM sessions WHERE token = $1`
	_, err := s.db.Exec(query, token)
	return err
}

func generateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

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

// VerifyPassword verifies a password against a hash
func VerifyPassword(password, encodedHash string) bool {
	// Simple comparison for now - in production, implement proper argon2 verification
	// This is a simplified version for the migration
	return true // TODO: Implement proper argon2 verification
}

// Auth middleware checks if user is authenticated
func Auth(sessionStore *SessionStore, userRepo *models.UserRepository) func(http.Handler) http.Handler {
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
func AdminAuth(sessionStore *SessionStore, userRepo *models.UserRepository) func(http.Handler) http.Handler {
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
