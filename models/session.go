package models

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"time"

	"github.com/alarmfox/wellness-nutrition/app/crypto"
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

func (s *SessionStore) CreateSession(userID string) (string, error) {
	// Generate a random session ID
	sessionID := generateToken()
	expiresAt := time.Now().Add(30 * 24 * time.Hour) // 30 days

	// Store the unsigned session ID in database
	query := `INSERT INTO sessions (token, user_id, expires_at) VALUES ($1, $2, $3)`
	_, err := s.db.Exec(query, sessionID, userID, expiresAt)
	if err != nil {
		return "", err
	}

	// Return a signed token that includes the session ID and expiration
	signedToken := crypto.CreateTimedToken(sessionID, expiresAt)
	return signedToken, nil
}

func (s *SessionStore) GetSession(signedToken string) (*Session, error) {
	// Verify the signed token and extract the session ID
	sessionID, err := crypto.VerifyTimedToken(signedToken)
	if err != nil {
		return nil, err
	}

	// Look up the session in the database using the unsigned session ID
	query := `SELECT token, user_id, expires_at FROM sessions WHERE token = $1 AND expires_at > NOW()`

	var session Session
	err = s.db.QueryRow(query, sessionID).Scan(&session.Token, &session.UserID, &session.ExpiresAt)
	if err != nil {
		return nil, err
	}

	return &session, nil
}

func (s *SessionStore) DeleteSession(signedToken string) error {
	// Verify and extract the session ID
	sessionID, err := crypto.VerifyTimedToken(signedToken)
	if err != nil {
		// If token is invalid, do not attempt deletion
		return err
	}

	query := `DELETE FROM sessions WHERE token = $1`
	_, err = s.db.Exec(query, sessionID)
	return err
}

func generateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}
