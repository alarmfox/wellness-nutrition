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
// Only allows truncating known test tables to prevent SQL injection
func TruncateTables(t *testing.T, db *sql.DB, tables ...string) {
	// Whitelist of allowed table names for testing
	allowedTables := map[string]bool{
		"users":       true,
		"sessions":    true,
		"bookings":    true,
		"events":      true,
		"instructors": true,
		"questions":   true,
	}

	for _, table := range tables {
		// Validate table name against whitelist
		if !allowedTables[table] {
			t.Logf("Warning: table %s is not in the allowed list, skipping", table)
			continue
		}

		// Safe to use fmt.Sprintf here since table name is validated
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
			last_name VARCHAR(255),
			address VARCHAR(255) NOT NULL,
			password TEXT,
			role VARCHAR(50) NOT NULL DEFAULT 'USER',
			med_ok BOOLEAN NOT NULL DEFAULT false,
			cellphone VARCHAR(50),
			sub_type VARCHAR(50) NOT NULL DEFAULT 'SHARED',
			email VARCHAR(255) NOT NULL UNIQUE,
			email_verified TIMESTAMPTZ,
			expires_at DATE NOT NULL,
			remaining_accesses INTEGER NOT NULL,
			verification_token VARCHAR(255),
			verification_token_expires_in TIMESTAMPTZ,
			goals TEXT,
			created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS instructors (
			id SERIAL PRIMARY KEY,
			first_name VARCHAR(255) NOT NULL,
			last_name VARCHAR(255) NOT NULL,
			max_slots INTEGER NOT NULL DEFAULT 2,
			created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS bookings (
			id BIGSERIAL PRIMARY KEY,
			user_id VARCHAR(255) REFERENCES users(id) ON DELETE CASCADE,
			instructor_id INTEGER NOT NULL REFERENCES instructors(id) ON DELETE CASCADE,
			starts_at TIMESTAMPTZ NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
			type VARCHAR(20) NOT NULL DEFAULT 'SIMPLE',
			CONSTRAINT unique_user_instructor_time UNIQUE (user_id, instructor_id, starts_at)
		);

		CREATE TABLE IF NOT EXISTS events (
			id SERIAL PRIMARY KEY,
			user_id VARCHAR(255) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			starts_at TIMESTAMPTZ NOT NULL,
			type VARCHAR(50) NOT NULL,
			occurred_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS sessions (
			token VARCHAR(255) PRIMARY KEY,
			user_id VARCHAR(255) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			expires_at TIMESTAMPTZ NOT NULL
		);

		CREATE TABLE IF NOT EXISTS questions (
			id SERIAL PRIMARY KEY,
			sku VARCHAR(255) UNIQUE NOT NULL,
			index INTEGER NOT NULL,
			next INTEGER NOT NULL,
			previous INTEGER NOT NULL,
			question TEXT NOT NULL,
			star1 INTEGER NOT NULL DEFAULT 0,
			star2 INTEGER NOT NULL DEFAULT 0,
			star3 INTEGER NOT NULL DEFAULT 0,
			star4 INTEGER NOT NULL DEFAULT 0,
			star5 INTEGER NOT NULL DEFAULT 0
		);
	`

	_, err := db.Exec(schema)
	if err != nil {
		t.Fatalf("Failed to create test schema: %v", err)
	}
}

// DropTestSchema drops all test tables
func DropTestSchema(t *testing.T, db *sql.DB) {
	tables := []string{"questions", "sessions", "bookings", "events", "instructors", "users"}

	for _, table := range tables {
		_, err := db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", table))
		if err != nil {
			t.Logf("Warning: failed to drop table %s: %v", table, err)
		}
	}
}
