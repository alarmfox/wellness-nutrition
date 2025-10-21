# Database Migrations

This directory contains SQL migration files for the Wellness & Nutrition application.

## Overview

All migrations are **idempotent** using `CREATE TABLE IF NOT EXISTS` statements, making them safe to run multiple times. The database schema uses **lowercase table and column names** with snake_case convention.

## Tables Created

### users
Main user table storing account information:
- `id` - Unique identifier (VARCHAR)
- `first_name`, `last_name` - User name
- `email` - Unique email address
- `password` - Hashed password (Argon2id)
- `role` - User role (USER or ADMIN)
- `sub_type` - Subscription type (SHARED or SINGLE)
- `expires_at` - Subscription expiration date
- `remaining_accesses` - Number of remaining bookings
- `email_verified` - Email verification timestamp
- And more...

### slots
Time slots for bookings:
- `starts_at` - Slot start time (PRIMARY KEY)
- `people_count` - Number of people booked
- `disabled` - Whether slot is disabled

### bookings
User bookings:
- `id` - Auto-increment ID
- `user_id` - Foreign key to users
- `starts_at` - Foreign key to slots
- `created_at` - Booking creation timestamp

### events
Event log for tracking booking actions:
- `id` - Auto-increment ID
- `user_id` - Foreign key to users
- `starts_at` - Event timestamp
- `type` - Event type (CREATED or DELETED)
- `occurred_at` - When event occurred

### sessions
User session management:
- `token` - Session token (PRIMARY KEY)
- `user_id` - Associated user
- `expires_at` - Session expiration

## Running Migrations

### Using the Migration Tool

```bash
cd app/migrations
go run . -db-uri="$DATABASE_URL"

# Or build and run
go build -o migrate
./migrate -db-uri="$DATABASE_URL"
```

### Manual Execution

You can also run the SQL files directly:

```bash
psql "$DATABASE_URL" < 001_initial_schema.sql
```

## Migration Files

- `001_initial_schema.sql` - Creates all base tables with indexes

## Features

- **Idempotent**: Safe to run multiple times
- **Lowercase names**: All table and column names use snake_case
- **Foreign keys**: Proper CASCADE constraints for data integrity
- **Indexes**: Optimized queries with indexes on frequently accessed columns
- **No Prisma dependency**: Pure SQL migrations

## Compatibility

The migration schema is designed to be compatible with:
- The existing Prisma schema (different naming convention)
- PostgreSQL 12+
- Direct SQL access from Go application

## Adding New Migrations

When adding new migrations:

1. Create a new file with incrementing number: `002_description.sql`
2. Use idempotent statements (`CREATE TABLE IF NOT EXISTS`, `CREATE INDEX IF NOT EXISTS`)
3. Use lowercase snake_case naming
4. Add foreign keys with CASCADE where appropriate
5. Add indexes for performance-critical queries

Example:

```sql
-- 002_add_feature.sql

-- New table
CREATE TABLE IF NOT EXISTS my_table (
    id SERIAL PRIMARY KEY,
    user_id VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- New index
CREATE INDEX IF NOT EXISTS idx_my_table_user_id ON my_table(user_id);
```

## Troubleshooting

### Connection Errors

Ensure your `DATABASE_URL` is correct:
```bash
export DATABASE_URL="******localhost:5432/wellness"
```

### Permission Errors

Ensure the database user has CREATE TABLE privileges:
```sql
GRANT CREATE ON DATABASE wellness TO myuser;
```

### Already Exists Errors

These are expected and safe with `IF NOT EXISTS` clauses. The migration tool will continue successfully.

## Notes

- Migrations are embedded in the Go binary using `//go:embed`
- The tool displays all tables after successful migration
- No migration tracking table is used (all statements are idempotent)
- Compatible with both new and existing databases
