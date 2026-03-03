package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
)

func main() {
	dbConnString := os.Getenv("DATABASE_URL")
	if dbConnString == "" {
		log.Fatal("database connection string is required (use DATABASE_URL env var)")
	}

	if len(os.Args) < 2 {
		log.Fatal("Usage: go run main.go <path_to_backup.sql>")
	}
	backupPath := os.Args[1]

	db, err := sql.Open("postgres", dbConnString)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}

	if err := runMigration(db, backupPath); err != nil {
		log.Printf("Migration failed: %v", err)
		os.Exit(1)
	}
	log.Println("Migration completed successfully.")
}

func runMigration(db *sql.DB, backupPath string) error {
	// 1. Read Backup File
	content, err := os.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("failed to read backup file: %w", err)
	}

	// 2. Prepare Temp Schema
	_, err = db.Exec("CREATE SCHEMA IF NOT EXISTS old_data;")
	if err != nil {
		return fmt.Errorf("failed to create old_data schema: %w", err)
	}

	// Ensure cleanup happens regardless of success/failure
	defer func() {
		_, _ = db.Exec("DROP SCHEMA IF EXISTS old_data CASCADE;")
	}()

	// 3. Load Data into Temp Schema
	_, err = db.Exec(string(content))
	if err != nil {
		return fmt.Errorf("failed to load backup into old_data: %w", err)
	}
	log.Println("Backup loaded into old_data schema.")
	// 4. Transform and Move Data in a Transaction
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()
	// Ensure a default instructor exists
	_, err = tx.Exec(`
    INSERT INTO public.instructors (id, first_name, last_name)
    VALUES (1, 'System', 'Default')
    ON CONFLICT (id) DO NOTHING`)
	if err != nil {
		return fmt.Errorf("failed to create default instructor: %w", err)
	}
	migrationTasks := []struct {
		name  string
		query string
	}{
		{
			name: "Users",
			query: `
        INSERT INTO public.users (
          id, first_name, last_name, address, password, role,
          med_ok, cellphone, sub_type, email, email_verified,
          expires_at, remaining_accesses, verification_token,
          verification_token_expires_in, goals
        )
        SELECT
          id, "firstName", "lastName", address, password, role::text,
          "medOk", cellphone, "subType"::text, email, "emailVerified",
          "expiresAt", "remainingAccesses", "verificationToken",
          "verificationTokenExpiresIn", goals
        FROM old_data."User"
        ON CONFLICT (email) DO NOTHING;`,
		},
		{
			name: "Events",
			query: `
        INSERT INTO public.events (user_id, starts_at, type, occurred_at)
        SELECT "userId", "startsAt", type::text, "occurredAt"
        FROM old_data."Event";`,
		},
		{
			name: "Bookings",
			query: `
        INSERT INTO public.bookings (user_id, instructor_id, starts_at, created_at, type)
        SELECT "userId", 1, "startsAt", "createdAt", 'SIMPLE'
        FROM old_data."Booking"
        ON CONFLICT (user_id, instructor_id, starts_at) DO NOTHING;`,
		},
		{
			name: "Sync Sequences",
			query: `
        SELECT setval('public.bookings_id_seq', coalesce((SELECT MAX(id) FROM public.bookings), 1));
        SELECT setval('public.events_id_seq', coalesce((SELECT MAX(id) FROM public.events), 1));
        SELECT setval('public.instructors_id_seq', coalesce((SELECT MAX(id) FROM public.instructors), 1));`,
		},
	}
	for _, task := range migrationTasks {
		log.Printf("Running migration task: %s", task.name)
		if _, err := tx.Exec(task.query); err != nil {
			return fmt.Errorf("error during %s migration: %w", task.name, err)
		}
	}
	return tx.Commit()
}
