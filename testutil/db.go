package testutil

import (
	"database/sql"
	"fmt"
	"os"
	"testing"

	_ "github.com/lib/pq"
)

// SetupTestDB creates a test database connection and returns it
// It will skip the test if DATABASE_URL is not set or if running with -short flag
func SetupTestDB(t *testing.T) *sql.DB {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	if err := db.Ping(); err != nil {
		t.Fatalf("Failed to ping test database: %v", err)
	}

	return db
}

// CleanupTestDB closes the database connection and cleans up test data
func CleanupTestDB(t *testing.T, db *sql.DB) {
	if db != nil {
		db.Close()
	}
}

// TruncateTables removes all data from specified tables
func TruncateTables(t *testing.T, db *sql.DB, tables ...string) {
	for _, table := range tables {
		_, err := db.Exec(fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table))
		if err != nil {
			t.Logf("Warning: failed to truncate table %s: %v", table, err)
		}
	}
}

// CreateTestSchema creates the database schema for testing
func CreateTestSchema(t *testing.T, db *sql.DB) {
	schema := `
		CREATE TABLE IF NOT EXISTS users (
			id VARCHAR(255) PRIMARY KEY,
			first_name VARCHAR(255) NOT NULL,
			last_name VARCHAR(255) NOT NULL,
			address TEXT,
			password VARCHAR(255),
			role VARCHAR(50) NOT NULL DEFAULT 'USER',
			med_ok BOOLEAN DEFAULT FALSE,
			cellphone VARCHAR(50),
			sub_type VARCHAR(50) DEFAULT 'SINGLE',
			email VARCHAR(255) UNIQUE NOT NULL,
			email_verified TIMESTAMP,
			expires_at TIMESTAMP NOT NULL,
			remaining_accesses INTEGER DEFAULT 0,
			verification_token VARCHAR(255),
			verification_token_expires_in TIMESTAMP,
			goals TEXT
		);

		CREATE TABLE IF NOT EXISTS sessions (
			token VARCHAR(255) PRIMARY KEY,
			user_id VARCHAR(255) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			expires_at TIMESTAMP NOT NULL
		);

		CREATE TABLE IF NOT EXISTS instructors (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			description TEXT
		);

		CREATE TABLE IF NOT EXISTS bookings (
			id SERIAL PRIMARY KEY,
			user_id VARCHAR(255) REFERENCES users(id) ON DELETE CASCADE,
			instructor_id INTEGER NOT NULL REFERENCES instructors(id),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			starts_at TIMESTAMP NOT NULL,
			type VARCHAR(50) DEFAULT 'SIMPLE'
		);

		CREATE TABLE IF NOT EXISTS events (
			id SERIAL PRIMARY KEY,
			instructor_id INTEGER NOT NULL REFERENCES instructors(id),
			starts_at TIMESTAMP NOT NULL,
			ends_at TIMESTAMP NOT NULL
		);

		CREATE TABLE IF NOT EXISTS questions (
			id SERIAL PRIMARY KEY,
			user_id VARCHAR(255) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			question TEXT NOT NULL,
			answer TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`

	_, err := db.Exec(schema)
	if err != nil {
		t.Fatalf("Failed to create test schema: %v", err)
	}
}

// DropTestSchema drops all test tables
func DropTestSchema(t *testing.T, db *sql.DB) {
	tables := []string{"questions", "bookings", "events", "sessions", "instructors", "users"}
	for _, table := range tables {
		_, err := db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", table))
		if err != nil {
			t.Logf("Warning: failed to drop table %s: %v", table, err)
		}
	}
}
