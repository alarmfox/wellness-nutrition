-- Migration: Add max_slots column to instructors table
ALTER TABLE instructors ADD COLUMN IF NOT EXISTS max_slots INTEGER NOT NULL DEFAULT 2;
