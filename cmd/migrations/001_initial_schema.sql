-- Migration: Create base tables
-- This migration creates all tables required for the application
-- All statements are idempotent (CREATE TABLE IF NOT EXISTS)

-- Users table (regular users only, no role column)
CREATE TABLE IF NOT EXISTS users (
    id VARCHAR(255) PRIMARY KEY,
    first_name VARCHAR(255) NOT NULL,
    last_name VARCHAR(255),
    address VARCHAR(255) NOT NULL,
    password TEXT,
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

-- Admins table (separate table for admin users)
CREATE TABLE IF NOT EXISTS admins (
    id VARCHAR(255) PRIMARY KEY,
    first_name VARCHAR(255) NOT NULL,
    last_name VARCHAR(255),
    email VARCHAR(255) NOT NULL UNIQUE,
    password TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Index on email for faster lookups
CREATE INDEX IF NOT EXISTS idx_admins_email ON admins(email);

-- Instructors table (instructors are just tags, no email/password)
CREATE TABLE IF NOT EXISTS instructors (
    id VARCHAR(255) PRIMARY KEY,
    first_name VARCHAR(255) NOT NULL,
    last_name VARCHAR(255),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Slots table (global time slots for bookings)
CREATE TABLE IF NOT EXISTS slots (
    starts_at TIMESTAMP PRIMARY KEY,
    people_count INTEGER NOT NULL DEFAULT 0,
    disabled BOOLEAN NOT NULL DEFAULT false
);

-- Bookings table
CREATE TABLE IF NOT EXISTS bookings (
    id BIGSERIAL PRIMARY KEY,
    user_id VARCHAR(255) NOT NULL,
    instructor_id VARCHAR(255),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    starts_at TIMESTAMP NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (instructor_id) REFERENCES instructors(id) ON DELETE SET NULL,
    FOREIGN KEY (starts_at) REFERENCES slots(starts_at) ON DELETE CASCADE
);

-- Indexes for bookings
CREATE INDEX IF NOT EXISTS idx_bookings_user_id ON bookings(user_id);
CREATE INDEX IF NOT EXISTS idx_bookings_starts_at ON bookings(starts_at);
CREATE INDEX IF NOT EXISTS idx_bookings_instructor_id ON bookings(instructor_id);

-- Index for slots
CREATE INDEX IF NOT EXISTS idx_slots_starts_at ON slots(starts_at);

-- Instructor slots table for per-instructor capacity tracking
CREATE TABLE IF NOT EXISTS instructor_slots (
    instructor_id VARCHAR(255) NOT NULL,
    starts_at TIMESTAMP NOT NULL,
    people_count INTEGER NOT NULL DEFAULT 0,
    max_capacity INTEGER NOT NULL DEFAULT 2,
    PRIMARY KEY (instructor_id, starts_at),
    FOREIGN KEY (instructor_id) REFERENCES instructors(id) ON DELETE CASCADE
);

-- Indexes for instructor_slots
CREATE INDEX IF NOT EXISTS idx_instructor_slots_instructor_id ON instructor_slots(instructor_id);
CREATE INDEX IF NOT EXISTS idx_instructor_slots_starts_at ON instructor_slots(starts_at);

-- Events table
CREATE TABLE IF NOT EXISTS events (
    id SERIAL PRIMARY KEY,
    user_id VARCHAR(255),
    admin_id VARCHAR(255),
    starts_at TIMESTAMP NOT NULL,
    type VARCHAR(50) NOT NULL,
    occurred_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (admin_id) REFERENCES admins(id) ON DELETE CASCADE
);

-- Index for events
CREATE INDEX IF NOT EXISTS idx_events_user_id ON events(user_id);

-- Sessions table (supports both users and admins)
CREATE TABLE IF NOT EXISTS sessions (
    token VARCHAR(255) PRIMARY KEY,
    user_id VARCHAR(255),
    admin_id VARCHAR(255),
    expires_at TIMESTAMP NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (admin_id) REFERENCES admins(id) ON DELETE CASCADE,
    CHECK ((user_id IS NOT NULL AND admin_id IS NULL) OR (user_id IS NULL AND admin_id IS NOT NULL))
);

-- Index on user_id for faster session lookups
CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id);

-- Index on admin_id for faster session lookups
CREATE INDEX IF NOT EXISTS idx_sessions_admin_id ON sessions(admin_id);

-- Index on expires_at for cleanup queries
CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions(expires_at);

-- Questions table for survey
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

-- Index on sku for faster lookups
CREATE INDEX IF NOT EXISTS idx_questions_sku ON questions(sku);

-- Index on index for ordering
CREATE INDEX IF NOT EXISTS idx_questions_index ON questions(index);
