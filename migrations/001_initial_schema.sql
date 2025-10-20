-- Migration: Create base tables
-- This migration creates all tables required for the application
-- All statements are idempotent (CREATE TABLE IF NOT EXISTS)

-- Users table
CREATE TABLE IF NOT EXISTS users (
    id VARCHAR(255) PRIMARY KEY,
    first_name VARCHAR(255) NOT NULL,
    last_name VARCHAR(255) NOT NULL,
    address VARCHAR(255) NOT NULL,
    password TEXT,
    role VARCHAR(50) NOT NULL DEFAULT 'USER',
    med_ok BOOLEAN NOT NULL DEFAULT false,
    cellphone VARCHAR(50),
    sub_type VARCHAR(50) NOT NULL DEFAULT 'SHARED',
    email VARCHAR(255) NOT NULL UNIQUE,
    email_verified TIMESTAMP,
    expires_at TIMESTAMP NOT NULL,
    remaining_accesses INTEGER NOT NULL,
    verification_token VARCHAR(255),
    verification_token_expires_in TIMESTAMP,
    goals TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Index on verification token for faster lookups
CREATE INDEX IF NOT EXISTS idx_users_verification_token ON users(verification_token);

-- Index on email for faster lookups
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);

-- Slots table
CREATE TABLE IF NOT EXISTS slots (
    starts_at TIMESTAMP PRIMARY KEY,
    people_count INTEGER NOT NULL DEFAULT 0,
    disabled BOOLEAN NOT NULL DEFAULT false
);

-- Bookings table
CREATE TABLE IF NOT EXISTS bookings (
    id BIGSERIAL PRIMARY KEY,
    user_id VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    starts_at TIMESTAMP NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (starts_at) REFERENCES slots(starts_at) ON DELETE CASCADE
);

-- Indexes for bookings
CREATE INDEX IF NOT EXISTS idx_bookings_user_id ON bookings(user_id);
CREATE INDEX IF NOT EXISTS idx_bookings_starts_at ON bookings(starts_at);

-- Events table
CREATE TABLE IF NOT EXISTS events (
    id SERIAL PRIMARY KEY,
    user_id VARCHAR(255) NOT NULL,
    starts_at TIMESTAMP NOT NULL,
    type VARCHAR(50) NOT NULL,
    occurred_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Index for events
CREATE INDEX IF NOT EXISTS idx_events_user_id ON events(user_id);

-- Sessions table
CREATE TABLE IF NOT EXISTS sessions (
    token VARCHAR(255) PRIMARY KEY,
    user_id VARCHAR(255) NOT NULL,
    expires_at TIMESTAMP NOT NULL
);

-- Index on user_id for faster session lookups
CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id);

-- Index on expires_at for cleanup queries
CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions(expires_at);
