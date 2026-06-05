package main

import (
	"database/sql"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/alarmfox/wellness-nutrition/app/testutil"
	_ "github.com/lib/pq"
)

func TestRunMigrationIntegration(t *testing.T) {
	oldDB, targetDB := setupMigrationTestDBs(t)
	defer oldDB.Close()
	defer targetDB.Close()

	createOldPublicSchema(t, oldDB)
	testutil.CreateTestSchema(t, targetDB)
	insertOldFixture(t, oldDB)

	report, err := runMigration(oldDB, targetDB)
	if err != nil {
		t.Fatalf("runMigration failed: %v", err)
	}

	if report.UsersInserted != 2 {
		t.Fatalf("expected 2 users inserted, got %d", report.UsersInserted)
	}
	if report.InstructorsInserted != 1 {
		t.Fatalf("expected 1 instructor inserted, got %d", report.InstructorsInserted)
	}
	if report.BookingsInserted != 1 {
		t.Fatalf("expected 1 booking inserted, got %d", report.BookingsInserted)
	}
	if report.BookingsSkipped != 1 {
		t.Fatalf("expected 1 duplicate booking skipped, got %d", report.BookingsSkipped)
	}
	if report.DisabledSlotBookingsSkipped != 1 {
		t.Fatalf("expected 1 disabled-slot booking skipped, got %d", report.DisabledSlotBookingsSkipped)
	}
	if report.DisabledSlotsInserted != 1 {
		t.Fatalf("expected 1 disabled slot inserted, got %d", report.DisabledSlotsInserted)
	}
	if report.EventsInserted != 1 {
		t.Fatalf("expected 1 event inserted, got %d", report.EventsInserted)
	}
	if report.QuestionsInserted != 1 {
		t.Fatalf("expected 1 question inserted, got %d", report.QuestionsInserted)
	}
	if report.SlotStats.Total != 2 || report.SlotStats.Disabled != 1 || report.SlotStats.NonDisabledWithPeople != 1 || report.SlotStats.NotRepresentedBookings != 0 {
		t.Fatalf("unexpected slot stats: %+v", report.SlotStats)
	}

	assertCount(t, targetDB, "public.users", 2)
	assertCount(t, targetDB, "public.instructors", 1)
	assertCount(t, targetDB, "public.bookings", 2)
	assertCount(t, targetDB, "public.events", 1)
	assertCount(t, targetDB, "public.questions", 1)

	var bookingType string
	var instructorID int64
	if err := targetDB.QueryRow(`SELECT type, instructor_id FROM public.bookings WHERE id = 42`).Scan(&bookingType, &instructorID); err != nil {
		t.Fatalf("failed to query migrated booking: %v", err)
	}
	if bookingType != "SIMPLE" || instructorID != 1 {
		t.Fatalf("unexpected migrated booking type/instructor: %s/%d", bookingType, instructorID)
	}

	var disabledUserID sql.NullString
	var disabledInstructorID int64
	var disabledStartsAt time.Time
	expectedDisabledStartsAt := time.Date(2026, 6, 10, 10, 0, 0, 0, time.UTC)
	if err := targetDB.QueryRow(`
		SELECT user_id, instructor_id, starts_at
		FROM public.bookings
		WHERE type = 'DISABLE'`).Scan(&disabledUserID, &disabledInstructorID, &disabledStartsAt); err != nil {
		t.Fatalf("failed to query migrated disabled slot: %v", err)
	}
	if disabledUserID.Valid || disabledInstructorID != 1 || !disabledStartsAt.Equal(expectedDisabledStartsAt) {
		t.Fatalf("unexpected migrated disabled slot: user=%v instructor=%d starts_at=%s", disabledUserID, disabledInstructorID, disabledStartsAt)
	}

	var nextBookingID int64
	if err := targetDB.QueryRow(`SELECT nextval('public.bookings_id_seq')`).Scan(&nextBookingID); err != nil {
		t.Fatalf("failed to query bookings sequence: %v", err)
	}
	if nextBookingID <= 42 {
		t.Fatalf("expected bookings sequence to advance past 42, got %d", nextBookingID)
	}
}

func TestRunMigrationFailsWhenTargetIsNotEmpty(t *testing.T) {
	oldDB, targetDB := setupMigrationTestDBs(t)
	defer oldDB.Close()
	defer targetDB.Close()

	createOldPublicSchema(t, oldDB)
	testutil.CreateTestSchema(t, targetDB)
	insertOldFixture(t, oldDB)

	_, err := targetDB.Exec(`
		INSERT INTO public.users (id, first_name, last_name, address, role, med_ok, sub_type, email, expires_at, remaining_accesses)
		VALUES ('existing', 'Existing', 'User', 'Address', 'USER', false, 'SHARED', 'existing@example.com', now(), 1)`)
	if err != nil {
		t.Fatalf("failed to insert existing target user: %v", err)
	}

	_, err = runMigration(oldDB, targetDB)
	if err == nil || !strings.Contains(err.Error(), "is not empty") {
		t.Fatalf("expected target-not-empty error, got %v", err)
	}
}

func setupMigrationTestDBs(t *testing.T) (*sql.DB, *sql.DB) {
	t.Helper()

	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	baseURL := os.Getenv("DATABASE_URL")
	if baseURL == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
	}

	adminURL := databaseURLWithName(t, baseURL, "postgres")
	adminDB, err := sql.Open("postgres", adminURL)
	if err != nil {
		t.Fatalf("failed to connect to admin database: %v", err)
	}
	if err := adminDB.Ping(); err != nil {
		adminDB.Close()
		t.Fatalf("failed to ping admin database: %v", err)
	}

	oldName := "nextjs2go_test_old"
	targetName := "nextjs2go_test_target"
	recreateDatabase(t, adminDB, oldName)
	recreateDatabase(t, adminDB, targetName)

	t.Cleanup(func() {
		dropDatabase(t, adminDB, oldName)
		dropDatabase(t, adminDB, targetName)
		adminDB.Close()
	})

	oldDB := openDatabase(t, databaseURLWithName(t, baseURL, oldName))
	targetDB := openDatabase(t, databaseURLWithName(t, baseURL, targetName))
	return oldDB, targetDB
}

func databaseURLWithName(t *testing.T, raw, dbName string) string {
	t.Helper()
	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("invalid DATABASE_URL: %v", err)
	}
	parsed.Path = "/" + dbName
	return parsed.String()
}

func recreateDatabase(t *testing.T, db *sql.DB, name string) {
	t.Helper()
	dropDatabase(t, db, name)
	if _, err := db.Exec("CREATE DATABASE " + pqQuoteIdentifier(name)); err != nil {
		t.Fatalf("failed to create database %s: %v", name, err)
	}
}

func dropDatabase(t *testing.T, db *sql.DB, name string) {
	t.Helper()
	_, err := db.Exec("DROP DATABASE IF EXISTS " + pqQuoteIdentifier(name) + " WITH (FORCE)")
	if err != nil {
		t.Fatalf("failed to drop database %s: %v", name, err)
	}
}

func openDatabase(t *testing.T, dbURL string) *sql.DB {
	t.Helper()
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("failed to open database %s: %v", dbURL, err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		t.Fatalf("failed to ping database %s: %v", dbURL, err)
	}
	return db
}

func createOldPublicSchema(t *testing.T, db *sql.DB) {
	t.Helper()
	_, err := db.Exec(`
		CREATE TYPE public."Role" AS ENUM ('ADMIN', 'USER');
		CREATE TYPE public."SubType" AS ENUM ('SHARED', 'SINGLE');
		CREATE TYPE public."EventType" AS ENUM ('CREATED', 'DELETED');

		CREATE TABLE public."User" (
			id text PRIMARY KEY,
			"firstName" text NOT NULL,
			"lastName" text NOT NULL,
			address text NOT NULL,
			password text,
			role public."Role" NOT NULL DEFAULT 'USER',
			"medOk" boolean NOT NULL DEFAULT false,
			cellphone text,
			"subType" public."SubType" NOT NULL DEFAULT 'SHARED',
			email text NOT NULL UNIQUE,
			"emailVerified" timestamp,
			"expiresAt" timestamp NOT NULL,
			"remainingAccesses" integer NOT NULL,
			"verificationToken" text,
			"verificationTokenExpiresIn" timestamp,
			goals text
		);

		CREATE TABLE public."Booking" (
			id bigint PRIMARY KEY,
			"userId" text NOT NULL REFERENCES public."User"(id),
			"createdAt" timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
			"startsAt" timestamp NOT NULL
		);

		CREATE TABLE public."Event" (
			id integer PRIMARY KEY,
			"userId" text NOT NULL REFERENCES public."User"(id),
			"startsAt" timestamp NOT NULL,
			type public."EventType" NOT NULL,
			"occurredAt" timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE public."Slot" (
			"startsAt" timestamp PRIMARY KEY,
			"peopleCount" integer NOT NULL DEFAULT 0,
			disabled boolean NOT NULL DEFAULT false
		);

		CREATE TABLE public.questions (
			id integer PRIMARY KEY,
			sku varchar(255) UNIQUE NOT NULL,
			index integer NOT NULL,
			next integer NOT NULL,
			previous integer NOT NULL,
			question text NOT NULL,
			star1 integer NOT NULL DEFAULT 0,
			star2 integer NOT NULL DEFAULT 0,
			star3 integer NOT NULL DEFAULT 0,
			star4 integer NOT NULL DEFAULT 0,
			star5 integer NOT NULL DEFAULT 0
		);
	`)
	if err != nil {
		t.Fatalf("failed to create old public schema: %v", err)
	}
}

func insertOldFixture(t *testing.T, db *sql.DB) {
	t.Helper()
	expiresAt := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	startsAt := time.Date(2026, 6, 10, 9, 0, 0, 0, time.UTC)
	createdAt := time.Date(2026, 6, 1, 8, 0, 0, 0, time.UTC)

	_, err := db.Exec(`
		INSERT INTO public."User" (id, "firstName", "lastName", address, password, role, "medOk", cellphone, "subType", email, "emailVerified", "expiresAt", "remainingAccesses", "verificationToken", "verificationTokenExpiresIn", goals)
		VALUES
			('admin', 'Admin', 'User', 'Office', NULL, 'ADMIN', true, NULL, 'SHARED', 'admin@example.com', NULL, $1, 99, NULL, NULL, NULL),
			('user-1', 'Regular', 'User', 'Home', 'hash', 'USER', false, '123', 'SINGLE', 'user@example.com', NULL, $1, 5, 'token', $1, 'Posturale')`,
		expiresAt)
	if err != nil {
		t.Fatalf("failed to insert old users: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO public."Booking" (id, "userId", "createdAt", "startsAt")
		VALUES
			(42, 'user-1', $1, $2),
			(43, 'user-1', $1, $2),
			(44, 'admin', $1, $2 + interval '1 hour')`,
		createdAt, startsAt)
	if err != nil {
		t.Fatalf("failed to insert old bookings: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO public."Event" (id, "userId", "startsAt", type, "occurredAt")
		VALUES (9, 'user-1', $1, 'CREATED', $2)`,
		startsAt, createdAt)
	if err != nil {
		t.Fatalf("failed to insert old events: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO public."Slot" ("startsAt", "peopleCount", disabled)
		VALUES ($1, 1, false), ($1 + interval '1 hour', 0, true)`,
		startsAt)
	if err != nil {
		t.Fatalf("failed to insert old slots: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO public.questions (id, sku, index, next, previous, question, star1, star2, star3, star4, star5)
		VALUES (7, 'step-1', 1, 0, 0, 'Question?', 1, 2, 3, 4, 5)`)
	if err != nil {
		t.Fatalf("failed to insert old questions: %v", err)
	}
}

func assertCount(t *testing.T, db *sql.DB, table string, want int64) {
	t.Helper()
	var got int64
	if err := db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&got); err != nil {
		t.Fatalf("failed to count %s: %v", table, err)
	}
	if got != want {
		t.Fatalf("expected %s count %d, got %d", table, want, got)
	}
}

func pqQuoteIdentifier(identifier string) string {
	return `"` + strings.ReplaceAll(identifier, `"`, `""`) + `"`
}

func TestDatabaseURLWithName(t *testing.T) {
	got := databaseURLWithName(t, "postgres://postgres:test123@localhost:5433/test_db?sslmode=disable", "postgres")
	if !strings.Contains(got, "/postgres?") {
		t.Fatalf("expected rewritten database name, got %s", got)
	}
}

func TestPQQuoteIdentifier(t *testing.T) {
	got := pqQuoteIdentifier(`bad"name`)
	if got != `"bad""name"` {
		t.Fatalf("unexpected quoted identifier: %s", got)
	}
}
