package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/lib/pq"
)

type migrationReport struct {
	UsersInserted               int64
	InstructorsInserted         int64
	BookingsInserted            int64
	BookingsSkipped             int64
	DisabledSlotBookingsSkipped int64
	DisabledSlotsInserted       int64
	EventsInserted              int64
	QuestionsInserted           int64
	SlotStats                   slotStats
}

type slotStats struct {
	Total                  int64
	Disabled               int64
	NonDisabledWithPeople  int64
	NotRepresentedBookings int64
}

func main() {
	oldURL := os.Getenv("OLD_DATABASE_URL")
	if oldURL == "" {
		log.Fatal("old database connection string is required (use OLD_DATABASE_URL env var)")
	}

	targetURL := os.Getenv("NEW_DATABASE_URL")
	if targetURL == "" {
		log.Fatal("target database connection string is required (use DATABASE_URL env var)")
	}
	fmt.Println(targetURL)

	oldDB, err := sql.Open("postgres", oldURL)
	if err != nil {
		log.Fatal(err)
	}
	defer oldDB.Close()

	targetDB, err := sql.Open("postgres", targetURL)
	if err != nil {
		log.Fatal(err)
	}
	defer targetDB.Close()

	if err := oldDB.Ping(); err != nil {
		log.Fatalf("failed to ping old database: %v", err)
	}
	if err := targetDB.Ping(); err != nil {
		log.Fatalf("failed to ping target database: %v", err)
	}

	report, err := runMigration(oldDB, targetDB)
	if err != nil {
		log.Printf("Migration failed: %v", err)
		os.Exit(1)
	}

	log.Println("Migration completed successfully.")
	log.Printf("Inserted users=%d instructors=%d bookings=%d disabled_slots=%d events=%d questions=%d",
		report.UsersInserted,
		report.InstructorsInserted,
		report.BookingsInserted,
		report.DisabledSlotsInserted,
		report.EventsInserted,
		report.QuestionsInserted,
	)
	if report.BookingsSkipped > 0 {
		log.Printf("Skipped duplicate old bookings=%d", report.BookingsSkipped)
	}
	if report.DisabledSlotBookingsSkipped > 0 {
		log.Printf("Skipped old bookings on disabled slots=%d", report.DisabledSlotBookingsSkipped)
	}
	log.Printf("Old slot summary: total=%d disabled=%d non_disabled_with_people=%d not_represented_by_booking=%d",
		report.SlotStats.Total,
		report.SlotStats.Disabled,
		report.SlotStats.NonDisabledWithPeople,
		report.SlotStats.NotRepresentedBookings,
	)
}

func runMigration(oldDB, targetDB *sql.DB) (*migrationReport, error) {
	if err := verifyOldSchema(oldDB); err != nil {
		return nil, err
	}
	if err := verifyTargetSchema(targetDB); err != nil {
		return nil, err
	}
	if err := verifyTargetEmpty(targetDB); err != nil {
		return nil, err
	}
	if err := verifyOldForeignKeys(oldDB); err != nil {
		return nil, err
	}

	report := &migrationReport{}
	stats, err := collectSlotStats(oldDB)
	if err != nil {
		return nil, err
	}
	report.SlotStats = stats

	tx, err := targetDB.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to start target transaction: %w", err)
	}
	defer tx.Rollback()

	instructorID, err := insertDefaultInstructor(tx)
	if err != nil {
		return nil, err
	}
	report.InstructorsInserted = 1

	if report.UsersInserted, err = copyUsers(oldDB, tx); err != nil {
		return nil, err
	}
	if report.BookingsInserted, report.BookingsSkipped, report.DisabledSlotBookingsSkipped, err = copyBookings(oldDB, tx, instructorID); err != nil {
		return nil, err
	}
	if report.DisabledSlotsInserted, err = copyDisabledSlots(oldDB, tx, instructorID); err != nil {
		return nil, err
	}
	if report.EventsInserted, err = copyEvents(oldDB, tx); err != nil {
		return nil, err
	}
	if report.QuestionsInserted, err = copyQuestions(oldDB, tx); err != nil {
		return nil, err
	}
	if err := resetSequences(tx); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit target transaction: %w", err)
	}

	return report, nil
}

func verifyOldSchema(db *sql.DB) error {
	required := []string{"User", "Booking", "Event", "Slot", "questions"}
	for _, table := range required {
		exists, err := tableExists(db, "public", table)
		if err != nil {
			return fmt.Errorf("failed to check old table public.%s: %w", table, err)
		}
		if !exists {
			return fmt.Errorf("old table public.%s does not exist", table)
		}
	}
	return nil
}

func verifyTargetSchema(db *sql.DB) error {
	required := []string{"users", "instructors", "bookings", "events", "questions"}
	for _, table := range required {
		exists, err := tableExists(db, "public", table)
		if err != nil {
			return fmt.Errorf("failed to check target table public.%s: %w", table, err)
		}
		if !exists {
			return fmt.Errorf("target table public.%s does not exist; run migrations first", table)
		}
	}
	return nil
}

func tableExists(db *sql.DB, schema, table string) (bool, error) {
	var exists bool
	err := db.QueryRow(`
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.tables
			WHERE table_schema = $1 AND table_name = $2
		)`, schema, table).Scan(&exists)
	return exists, err
}

func verifyTargetEmpty(db *sql.DB) error {
	tables := []string{"users", "instructors", "bookings", "events", "questions"}
	for _, table := range tables {
		var count int64
		query := fmt.Sprintf("SELECT COUNT(*) FROM public.%s", pq.QuoteIdentifier(table))
		if err := db.QueryRow(query).Scan(&count); err != nil {
			return fmt.Errorf("failed to count target table public.%s: %w", table, err)
		}
		if count > 0 {
			return fmt.Errorf("target table public.%s is not empty (%d rows); use a fresh migrated database", table, count)
		}
	}
	return nil
}

func verifyOldForeignKeys(db *sql.DB) error {
	bookingMissing, err := countMissingUsers(db, "Booking")
	if err != nil {
		return err
	}
	eventMissing, err := countMissingUsers(db, "Event")
	if err != nil {
		return err
	}
	if bookingMissing > 0 || eventMissing > 0 {
		return fmt.Errorf("old database has missing user references: bookings=%d events=%d", bookingMissing, eventMissing)
	}
	return nil
}

func countMissingUsers(db *sql.DB, table string) (int64, error) {
	query := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM %s source
		LEFT JOIN %s users ON users.id = source."userId"
		WHERE users.id IS NULL`,
		publicTable(table),
		publicTable("User"),
	)
	var count int64
	if err := db.QueryRow(query).Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to validate old %s user references: %w", table, err)
	}
	return count, nil
}

func collectSlotStats(db *sql.DB) (slotStats, error) {
	query := fmt.Sprintf(`
		SELECT
			COUNT(*),
			COUNT(*) FILTER (WHERE disabled),
			COUNT(*) FILTER (WHERE NOT disabled AND "peopleCount" > 0),
			COUNT(*) FILTER (
				WHERE NOT EXISTS (
					SELECT 1
					FROM %s b
					WHERE b."startsAt" = s."startsAt"
				)
			)
		FROM %s s`,
		publicTable("Booking"),
		publicTable("Slot"),
	)

	var stats slotStats
	if err := db.QueryRow(query).Scan(
		&stats.Total,
		&stats.Disabled,
		&stats.NonDisabledWithPeople,
		&stats.NotRepresentedBookings,
	); err != nil {
		return slotStats{}, fmt.Errorf("failed to collect old slot stats: %w", err)
	}
	return stats, nil
}

func insertDefaultInstructor(tx *sql.Tx) (int64, error) {
	var id int64
	err := tx.QueryRow(`
		INSERT INTO public.instructors (first_name, last_name, max_slots)
		VALUES ('System', 'Default', 2)
		RETURNING id`).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("failed to create default instructor: %w", err)
	}
	return id, nil
}

func copyUsers(oldDB *sql.DB, tx *sql.Tx) (int64, error) {
	rows, err := oldDB.Query(fmt.Sprintf(`
		SELECT id, "firstName", "lastName", address, password, role::text, "medOk",
			cellphone, "subType"::text, email,
			"emailVerified" AT TIME ZONE 'UTC',
			"expiresAt"::date,
			"remainingAccesses", "verificationToken",
			"verificationTokenExpiresIn" AT TIME ZONE 'UTC',
			goals
		FROM %s
		ORDER BY id`, publicTable("User")))
	if err != nil {
		return 0, fmt.Errorf("failed to read old users: %w", err)
	}
	defer rows.Close()

	stmt, err := tx.Prepare(`
		INSERT INTO public.users (
			id, first_name, last_name, address, password, role,
			med_ok, cellphone, sub_type, email, email_verified,
			expires_at, remaining_accesses, verification_token,
			verification_token_expires_in, goals
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)`)
	if err != nil {
		return 0, fmt.Errorf("failed to prepare target user insert: %w", err)
	}
	defer stmt.Close()

	var inserted int64
	for rows.Next() {
		var user oldUser
		if err := rows.Scan(
			&user.ID,
			&user.FirstName,
			&user.LastName,
			&user.Address,
			&user.Password,
			&user.Role,
			&user.MedOK,
			&user.Cellphone,
			&user.SubType,
			&user.Email,
			&user.EmailVerified,
			&user.ExpiresAt,
			&user.RemainingAccesses,
			&user.VerificationToken,
			&user.VerificationTokenExpiresIn,
			&user.Goals,
		); err != nil {
			return 0, fmt.Errorf("failed to scan old user: %w", err)
		}

		if _, err := stmt.Exec(
			user.ID,
			user.FirstName,
			user.LastName,
			user.Address,
			user.Password,
			user.Role,
			user.MedOK,
			user.Cellphone,
			user.SubType,
			user.Email,
			user.EmailVerified,
			user.ExpiresAt,
			user.RemainingAccesses,
			user.VerificationToken,
			user.VerificationTokenExpiresIn,
			user.Goals,
		); err != nil {
			return 0, fmt.Errorf("failed to insert target user %s: %w", user.ID, err)
		}
		inserted++
	}

	return inserted, rows.Err()
}

func copyBookings(oldDB *sql.DB, tx *sql.Tx, instructorID int64) (int64, int64, int64, error) {
	disabledSlotBookings, err := countOldBookingsOnDisabledSlots(oldDB)
	if err != nil {
		return 0, 0, 0, err
	}

	rows, err := oldDB.Query(fmt.Sprintf(`
		SELECT
			b.id,
			b."userId",
			b."createdAt" AT TIME ZONE 'UTC',
			b."startsAt" AT TIME ZONE 'UTC'
		FROM %s b
		WHERE NOT EXISTS (
			SELECT 1
			FROM %s s
			WHERE s."startsAt" = b."startsAt"
				AND s.disabled
		)
		ORDER BY b.id`,
		publicTable("Booking"),
		publicTable("Slot"),
	))
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to read old bookings: %w", err)
	}
	defer rows.Close()

	stmt, err := tx.Prepare(`
		INSERT INTO public.bookings (id, user_id, instructor_id, starts_at, created_at, type)
		VALUES ($1, $2, $3, $4, $5, 'SIMPLE')
		ON CONFLICT (user_id, instructor_id, starts_at) DO NOTHING`)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to prepare target booking insert: %w", err)
	}
	defer stmt.Close()

	var inserted int64
	var skipped int64
	for rows.Next() {
		var id int64
		var userID string
		var createdAt, startsAt time.Time
		if err := rows.Scan(&id, &userID, &createdAt, &startsAt); err != nil {
			return 0, 0, 0, fmt.Errorf("failed to scan old booking: %w", err)
		}
		result, err := stmt.Exec(id, userID, instructorID, startsAt, createdAt)
		if err != nil {
			return 0, 0, 0, fmt.Errorf("failed to insert target booking %d: %w", id, err)
		}
		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return 0, 0, 0, fmt.Errorf("failed to inspect target booking insert %d: %w", id, err)
		}
		if rowsAffected == 0 {
			skipped++
			continue
		}
		inserted++
	}

	return inserted, skipped, disabledSlotBookings, rows.Err()
}

func countOldBookingsOnDisabledSlots(db *sql.DB) (int64, error) {
	query := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM %s b
		JOIN %s s ON s."startsAt" = b."startsAt"
		WHERE s.disabled`,
		publicTable("Booking"),
		publicTable("Slot"),
	)
	var count int64
	if err := db.QueryRow(query).Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to count old bookings on disabled slots: %w", err)
	}
	return count, nil
}

func copyDisabledSlots(oldDB *sql.DB, tx *sql.Tx, instructorID int64) (int64, error) {
	rows, err := oldDB.Query(fmt.Sprintf(`
		SELECT "startsAt" AT TIME ZONE 'UTC'
		FROM %s
		WHERE disabled
		ORDER BY "startsAt"`, publicTable("Slot")))
	if err != nil {
		return 0, fmt.Errorf("failed to read old disabled slots: %w", err)
	}
	defer rows.Close()

	stmt, err := tx.Prepare(`
		INSERT INTO public.bookings (user_id, instructor_id, starts_at, created_at, type)
		SELECT NULL, $1, $2, $2, 'DISABLE'
		WHERE NOT EXISTS (
			SELECT 1
			FROM public.bookings
			WHERE user_id IS NULL
				AND instructor_id = $1
				AND starts_at = $2
				AND type = 'DISABLE'
		)`)
	if err != nil {
		return 0, fmt.Errorf("failed to prepare target disabled slot insert: %w", err)
	}
	defer stmt.Close()

	var inserted int64
	for rows.Next() {
		var startsAt time.Time
		if err := rows.Scan(&startsAt); err != nil {
			return 0, fmt.Errorf("failed to scan old disabled slot: %w", err)
		}
		result, err := stmt.Exec(instructorID, startsAt)
		if err != nil {
			return 0, fmt.Errorf("failed to insert target disabled slot at %s: %w", startsAt.Format(time.RFC3339), err)
		}
		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return 0, fmt.Errorf("failed to inspect target disabled slot insert at %s: %w", startsAt.Format(time.RFC3339), err)
		}
		inserted += rowsAffected
	}

	return inserted, rows.Err()
}

func copyEvents(oldDB *sql.DB, tx *sql.Tx) (int64, error) {
	rows, err := oldDB.Query(fmt.Sprintf(`
		SELECT
			id,
			"userId",
			"startsAt" AT TIME ZONE 'UTC',
			type::text,
			"occurredAt" AT TIME ZONE 'UTC'
		FROM %s
		ORDER BY id`, publicTable("Event")))
	if err != nil {
		return 0, fmt.Errorf("failed to read old events: %w", err)
	}
	defer rows.Close()

	stmt, err := tx.Prepare(`
		INSERT INTO public.events (id, user_id, starts_at, type, occurred_at)
		VALUES ($1, $2, $3, $4, $5)`)
	if err != nil {
		return 0, fmt.Errorf("failed to prepare target event insert: %w", err)
	}
	defer stmt.Close()

	var inserted int64
	for rows.Next() {
		var id int64
		var userID, eventType string
		var startsAt, occurredAt time.Time
		if err := rows.Scan(&id, &userID, &startsAt, &eventType, &occurredAt); err != nil {
			return 0, fmt.Errorf("failed to scan old event: %w", err)
		}
		if _, err := stmt.Exec(id, userID, startsAt, eventType, occurredAt); err != nil {
			return 0, fmt.Errorf("failed to insert target event %d: %w", id, err)
		}
		inserted++
	}

	return inserted, rows.Err()
}

func copyQuestions(oldDB *sql.DB, tx *sql.Tx) (int64, error) {
	rows, err := oldDB.Query(fmt.Sprintf(`
		SELECT id, sku, index, next, previous, question, star1, star2, star3, star4, star5
		FROM %s
		ORDER BY id`, publicTable("questions")))
	if err != nil {
		return 0, fmt.Errorf("failed to read old questions: %w", err)
	}
	defer rows.Close()

	stmt, err := tx.Prepare(`
		INSERT INTO public.questions (id, sku, index, next, previous, question, star1, star2, star3, star4, star5)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`)
	if err != nil {
		return 0, fmt.Errorf("failed to prepare target question insert: %w", err)
	}
	defer stmt.Close()

	var inserted int64
	for rows.Next() {
		var q oldQuestion
		if err := rows.Scan(&q.ID, &q.SKU, &q.Index, &q.Next, &q.Previous, &q.Question, &q.Star1, &q.Star2, &q.Star3, &q.Star4, &q.Star5); err != nil {
			return 0, fmt.Errorf("failed to scan old question: %w", err)
		}
		if _, err := stmt.Exec(q.ID, q.SKU, q.Index, q.Next, q.Previous, q.Question, q.Star1, q.Star2, q.Star3, q.Star4, q.Star5); err != nil {
			return 0, fmt.Errorf("failed to insert target question %d: %w", q.ID, err)
		}
		inserted++
	}

	return inserted, rows.Err()
}

func resetSequences(tx *sql.Tx) error {
	sequences := map[string]string{
		"public.bookings_id_seq":    "public.bookings",
		"public.events_id_seq":      "public.events",
		"public.instructors_id_seq": "public.instructors",
		"public.questions_id_seq":   "public.questions",
	}

	for sequence, table := range sequences {
		parts := strings.Split(table, ".")
		if len(parts) != 2 {
			return fmt.Errorf("invalid table name %s", table)
		}
		query := fmt.Sprintf(
			"SELECT setval(%s, COALESCE((SELECT MAX(id) FROM %s.%s), 1), true)",
			pq.QuoteLiteral(sequence),
			pq.QuoteIdentifier(parts[0]),
			pq.QuoteIdentifier(parts[1]),
		)
		if _, err := tx.Exec(query); err != nil {
			return fmt.Errorf("failed to reset sequence %s: %w", sequence, err)
		}
	}
	return nil
}

func publicTable(table string) string {
	return "public." + pq.QuoteIdentifier(table)
}

type oldUser struct {
	ID                         string
	FirstName                  string
	LastName                   string
	Address                    string
	Password                   sql.NullString
	Role                       string
	MedOK                      bool
	Cellphone                  sql.NullString
	SubType                    string
	Email                      string
	EmailVerified              sql.NullTime
	ExpiresAt                  time.Time
	RemainingAccesses          int
	VerificationToken          sql.NullString
	VerificationTokenExpiresIn sql.NullTime
	Goals                      sql.NullString
}

type oldQuestion struct {
	ID       int
	SKU      string
	Index    int
	Next     int
	Previous int
	Question string
	Star1    int
	Star2    int
	Star3    int
	Star4    int
	Star5    int
}
