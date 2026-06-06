-- Migration: Add instructor availability settings
ALTER TABLE instructors ADD COLUMN IF NOT EXISTS max_slots INTEGER NOT NULL DEFAULT 2;
ALTER TABLE instructors ADD COLUMN IF NOT EXISTS enabled BOOLEAN NOT NULL DEFAULT TRUE;
