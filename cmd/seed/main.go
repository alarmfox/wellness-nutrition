package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
	"golang.org/x/crypto/argon2"
)

func main() {
	dbConnString := os.Getenv("DATABASE_URL")

	if dbConnString == "" {
		log.Fatal("database connection string is required (use DATABASE_URL env var)")
	}

	db, err := sql.Open("postgres", dbConnString)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}

	seedTest(db)
}

func seedTest(db *sql.DB) {
	var err error

	// Create admin user with ADMIN role
	adminPassword := hashPassword("admin123")
	adminID := generateID()
	_, err = db.Exec(`
		INSERT INTO users 
		(id, first_name, last_name, address, password, role, med_ok, email, email_verified, expires_at, remaining_accesses, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		ON CONFLICT (email) DO NOTHING
	`, adminID, "Admin", "User", "Admin Address", adminPassword, "ADMIN", true, "admin@wellness.local", time.Now(), time.Now().AddDate(1, 0, 0), 9999, time.Now(), time.Now())
	if err != nil {
		log.Printf("Warning: Could not create admin: %v", err)
	} else {
		log.Println("✓ Created admin user (email: admin@wellness.local, password: admin123, role: ADMIN)")
	}

	// Create test users
	users := []struct {
		firstName string
		lastName  string
		email     string
		subType   string
		accesses  int
	}{
		{"Mario", "Rossi", "mario.rossi@test.local", "SHARED", 10},
		{"Laura", "Bianchi", "laura.bianchi@test.local", "SINGLE", 8},
		{"Giuseppe", "Verdi", "giuseppe.verdi@test.local", "SHARED", 12},
		{"Anna", "Romano", "anna.romano@test.local", "SINGLE", 5},
		{"Francesco", "Ferrari", "francesco.ferrari@test.local", "SHARED", 15},
	}

	userPassword := hashPassword("password123")
	userIDs := make([]string, len(users))

	for i, u := range users {
		userID := generateID()
		userIDs[i] = userID
		_, err = db.Exec(`
			INSERT INTO users 
			(id, first_name, last_name, address, password, med_ok, 
			 sub_type, email, email_verified, expires_at, remaining_accesses)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
			ON CONFLICT (email) DO NOTHING
		`, userID, u.firstName, u.lastName, "Via Test "+u.lastName, userPassword, true,
			u.subType, u.email, time.Now(), time.Now().AddDate(0, 6, 0), u.accesses)
		if err != nil {
			log.Printf("Warning: Could not create user %s: %v", u.email, err)
		} else {
			log.Printf("✓ Created user %s %s (email: %s, password: password123)", u.firstName, u.lastName, u.email)
		}
	}

	// Create instructors (just tags, no email/password)
	instructors := []struct {
		firstName string
		lastName  string
	}{
		{"Marco", "Bianchi"},
		{"Giulia", "Ferrari"},
		{"Alessandro", "Russo"},
	}

	instructorIDs := make([]int, len(instructors))

	var id int
	for i, instr := range instructors {
		err := db.QueryRow(`
			INSERT INTO instructors 
			(first_name, last_name)
			VALUES ($1, $2)
			ON CONFLICT DO NOTHING
			RETURNING id
		`, instr.firstName, instr.lastName).Scan(&id)
		if err != nil {
			log.Printf("Warning: Could not create instructor %s %s: %v", instr.firstName, instr.lastName, err)
		} else {
			log.Printf("✓ Created instructor %s %s", instr.firstName, instr.lastName)
		}
		instructorIDs[i] = id
	}

	// Create time slots for the next 30 days
	// Load Europe/Rome timezone to determine DST correctly
	romeLocation, err := time.LoadLocation("Europe/Rome")
	if err != nil {
		log.Fatalf("Failed to load Europe/Rome timezone: %v", err)
	}

	now := time.Now().UTC()
	startDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	// Create some bookings for the test users
	var bookingsCreated int
	for i, userID := range userIDs {
		// Create 2-3 bookings per user
		numBookings := 2
		if i%2 == 0 {
			numBookings = 3
		}

		for j := range numBookings {
			// Book slots in the next week at different times
			bookingDay := 1 + (i*2 + j)  // Spread bookings across days
			bookingHour := 10 + (i*2)%10 // Different hours for each user

			// Create time at this hour in Rome timezone, then convert to UTC
			bookingTimeRome := time.Date(startDate.Year(), startDate.Month(), startDate.Day(),
				0, 0, 0, 0, romeLocation).AddDate(0, 0, bookingDay)
			bookingTime := time.Date(bookingTimeRome.Year(), bookingTimeRome.Month(), bookingTimeRome.Day(),
				bookingHour, 0, 0, 0, romeLocation).UTC()

			// Skip if Sunday
			if bookingTime.Weekday() == time.Sunday {
				continue
			}

			// Assign random instructor
			instructorID := instructorIDs[(i+j)%len(instructorIDs)]

			_, err = db.Exec(`
				INSERT INTO bookings (user_id, instructor_id, created_at, starts_at, type)
				VALUES ($1, $2, $3, $4, $5)
			`, userID, instructorID, time.Now().Add(-time.Duration(j)*24*time.Hour), bookingTime, "SIMPLE")
			if err != nil {
				log.Printf("Warning: Could not create booking: %v", err)
			} else {
				bookingsCreated++
			}
		}
	}
	log.Printf("✓ Created %d bookings for test users", bookingsCreated)

	log.Println("\nTest Accounts:")
	log.Println("  Admin: admin@wellness.local / admin123")
	log.Println("  Users: *.@test.local / password123")
	log.Println("    - mario.rossi@test.local")
	log.Println("    - laura.bianchi@test.local")
	log.Println("    - giuseppe.verdi@test.local")
	log.Println("    - anna.romano@test.local")
	log.Println("    - francesco.ferrari@test.local")

	log.Println("  Instructors (tags only, no login):")
	log.Println("    - Marco Bianchi")
	log.Println("    - Giulia Ferrari")
	log.Println("    - Alessandro Russo")

	log.Println("\nSeeding survey questions...")

	questions := []struct {
		sku      string
		index    int
		question string
	}{
		{"q1", 1, "Come giudichi la qualità del servizio ricevuto?"},
		{"q2", 2, "Quanto sei soddisfatto/a della professionalità dello staff?"},
		{"q3", 3, "Le informazioni ricevute sono state chiare e utili?"},
		{"q4", 4, "Come valuti l'ambiente e la pulizia della struttura?"},
		{"q5", 5, "Raccomanderesti questo servizio ad amici o familiari?"},
	}

	for i, q := range questions {
		previous := 0
		next := 0

		if i > 0 {
			previous = questions[i-1].index
		}
		if i < len(questions)-1 {
			next = questions[i+1].index
		}

		query := `INSERT INTO questions (sku, index, next, previous, question, star1, star2, star3, star4, star5)
  VALUES ($1, $2, $3, $4, $5, 0, 0, 0, 0, 0)
  ON CONFLICT (sku) DO NOTHING`

		_, err := db.Exec(query, q.sku, q.index, next, previous, q.question)
		if err != nil {
			log.Printf("Error seeding question %s: %v", q.sku, err)
		} else {
			log.Printf("Seeded question: %s", q.question)
		}
	}

	log.Println("Survey seeding completed!")
	log.Println("\n\n=== Seeding Complete ===")
}

func hashPassword(password string) string {
	salt := make([]byte, 16)
	rand.Read(salt)

	hash := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)

	encoded := "$argon2id$v=19$m=65536,t=1,p=4$" +
		base64.RawStdEncoding.EncodeToString(salt) + "$" +
		base64.RawStdEncoding.EncodeToString(hash)

	return encoded
}

func generateID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)[:22]
}
