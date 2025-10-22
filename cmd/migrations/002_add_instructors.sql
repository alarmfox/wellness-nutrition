-- Migration: Add instructors table and instructor_id to bookings
-- This migration adds support for per-instructor booking management

-- Create instructors table (instructors are just tags, no email/password)
CREATE TABLE IF NOT EXISTS instructors (
    id VARCHAR(255) PRIMARY KEY,
    first_name VARCHAR(255) NOT NULL,
    last_name VARCHAR(255),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Add instructor_id column to bookings table if it doesn't exist
DO $$ 
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'bookings' AND column_name = 'instructor_id'
    ) THEN
        ALTER TABLE bookings ADD COLUMN instructor_id VARCHAR(255);
    END IF;
END $$;

-- Add foreign key constraint for instructor_id
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints 
        WHERE constraint_name = 'fk_bookings_instructor_id'
    ) THEN
        ALTER TABLE bookings 
        ADD CONSTRAINT fk_bookings_instructor_id 
        FOREIGN KEY (instructor_id) REFERENCES instructors(id) ON DELETE SET NULL;
    END IF;
END $$;

-- Create index on instructor_id for faster filtering
CREATE INDEX IF NOT EXISTS idx_bookings_instructor_id ON bookings(instructor_id);

-- Create instructor_slots table for per-instructor capacity tracking
CREATE TABLE IF NOT EXISTS instructor_slots (
    instructor_id VARCHAR(255) NOT NULL,
    starts_at TIMESTAMP NOT NULL,
    people_count INTEGER NOT NULL DEFAULT 0,
    max_capacity INTEGER NOT NULL DEFAULT 2,
    PRIMARY KEY (instructor_id, starts_at),
    FOREIGN KEY (instructor_id) REFERENCES instructors(id) ON DELETE CASCADE
);

-- Index for faster lookups
CREATE INDEX IF NOT EXISTS idx_instructor_slots_instructor_id ON instructor_slots(instructor_id);
CREATE INDEX IF NOT EXISTS idx_instructor_slots_starts_at ON instructor_slots(starts_at);
