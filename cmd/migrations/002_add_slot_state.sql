-- Migration: Add state column to slots table
-- This migration adds a state column to track FREE, UNAVAILABLE, MASSAGE, and APPOINTMENT states
-- and migrates existing disabled data to the new state column

-- Add state column with default value 'FREE'
ALTER TABLE slots ADD COLUMN IF NOT EXISTS state VARCHAR(20) NOT NULL DEFAULT 'FREE';

-- Migrate existing disabled data to state
-- disabled = true -> state = 'UNAVAILABLE'
-- disabled = false -> state = 'FREE'
UPDATE slots SET state = 'UNAVAILABLE' WHERE disabled = true;
UPDATE slots SET state = 'FREE' WHERE disabled = false;

-- Note: We keep the disabled column for backward compatibility
-- In a future migration, we could remove it after verifying the migration is successful
