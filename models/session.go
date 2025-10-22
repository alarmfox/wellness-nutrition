package models

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"time"
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
