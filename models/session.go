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

// ExtendSession extends the expiration of an existing session
// Returns a new signed token with the updated expiration time
func (s *SessionStore) ExtendSession(signedToken string, newExpiresAt time.Time) (string, error) {
	// Verify and extract the session ID from the old token
	sessionID, err := crypto.VerifyTimedToken(signedToken)
	if err != nil {
		return "", err
	}

	// Update the session expiration in the database
	query := `UPDATE sessions SET expires_at = $1 WHERE token = $2 AND expires_at > NOW()`
	result, err := s.db.Exec(query, newExpiresAt, sessionID)
	if err != nil {
		return "", err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return "", err
	}
	if rowsAffected == 0 {
		return "", sql.ErrNoRows
	}

	// Return a new signed token with the updated expiration
	newSignedToken := crypto.CreateTimedToken(sessionID, newExpiresAt)
	return newSignedToken, nil
}

func generateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}
