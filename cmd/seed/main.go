package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"flag"
	"log"
	"os"
	"strings"
	"time"

	_ "github.com/lib/pq"
	"golang.org/x/crypto/argon2"
)

var cmd = flag.String("seed", "", "What to seed")

func main() {
	flag.Parse()
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

	switch strings.ToLower(*cmd) {
	// Seeds test data
	case "test":
		seedTest(db)
	// Pre-create slots
	case "slot":
		seedSlot(db)
	default:
		log.Fatalf("unknown command %q", *cmd)
	}
}

func seedTest(db *sql.DB) {
	var err error

	adminPassword := hashPassword("admin123")
	adminID := generateID()
	_, err = db.Exec(`
		INSERT INTO users 
		(id, first_name, last_name, address, password, role, med_ok, 
		 sub_type, email, email_verified, expires_at, remaining_accesses)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (email) DO NOTHING
	`, adminID, "Admin", "User", "123 Admin St", adminPassword, "ADMIN", true,
		"SINGLE", "admin@wellness.local", time.Now(), time.Now().AddDate(1, 0, 0), 999)
	if err != nil {
		log.Printf("Warning: Could not create admin user: %v", err)
	} else {
		log.Println("✓ Created admin user (email: admin@wellness.local, password: admin123)")
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
			(id, first_name, last_name, address, password, role, med_ok, 
			 sub_type, email, email_verified, expires_at, remaining_accesses)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
			ON CONFLICT (email) DO NOTHING
		`, userID, u.firstName, u.lastName, "Via Test "+u.lastName, userPassword, "USER", true,
			u.subType, u.email, time.Now(), time.Now().AddDate(0, 6, 0), u.accesses)
		if err != nil {
			log.Printf("Warning: Could not create user %s: %v", u.email, err)
		} else {
			log.Printf("✓ Created user %s %s (email: %s, password: password123)", u.firstName, u.lastName, u.email)
		}
	}

	// Create time slots for the next 30 days
	now := time.Now().UTC()
	startDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	var slotsCreated int
	for day := 0; day < 30; day++ {
		currentDate := startDate.AddDate(0, 0, day)

		// Skip Sundays (weekday 0)
		if currentDate.Weekday() == time.Sunday {
			continue
		}

		// Create slots from 07:00 to 21:00 (every hour)
		for hour := 7; hour <= 21; hour++ {
			slotTime := time.Date(currentDate.Year(), currentDate.Month(), currentDate.Day(),
				hour, 0, 0, 0, time.UTC)

			_, err = db.Exec(`
				INSERT INTO slots (starts_at, people_count, disabled)
				VALUES ($1, $2, $3)
				ON CONFLICT (starts_at) DO NOTHING
			`, slotTime, 0, false)
			if err != nil {
				log.Printf("Warning: Could not create slot at %v: %v", slotTime, err)
			} else {
				slotsCreated++
			}
		}
	}
	log.Printf("✓ Created %d time slots for the next 30 days", slotsCreated)

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

			bookingTime := startDate.AddDate(0, 0, bookingDay)
			bookingTime = time.Date(bookingTime.Year(), bookingTime.Month(), bookingTime.Day(),
				bookingHour, 0, 0, 0, time.UTC)

			// Skip if Sunday
			if bookingTime.Weekday() == time.Sunday {
				continue
			}

			_, err = db.Exec(`
				INSERT INTO bookings (user_id, created_at, starts_at)
				VALUES ($1, $2, $3)
			`, userID, time.Now().Add(-time.Duration(j)*24*time.Hour), bookingTime)
			if err != nil {
				log.Printf("Warning: Could not create booking: %v", err)
			} else {
				bookingsCreated++

				// Update slot people count
				_, _ = db.Exec(`
					UPDATE slots
					SET people_count = people_count + 1
					WHERE starts_at = $1
				`, bookingTime)
			}
		}
	}
	log.Printf("✓ Created %d bookings for test users", bookingsCreated)

	log.Println("\n=== Seeding Complete ===")
	log.Println("\nTest Accounts:")
	log.Println("  Admin: admin@wellness.local / admin123")
	log.Println("  Users: *.@test.local / password123")
	log.Println("    - mario.rossi@test.local")
	log.Println("    - laura.bianchi@test.local")
	log.Println("    - giuseppe.verdi@test.local")
	log.Println("    - anna.romano@test.local")
	log.Println("    - francesco.ferrari@test.local")
}

func seedSlot(db *sql.DB) {
	log.Print("Seeding slots for the next year... this may take some time")
	var (
		err          error
		slotsCreated = 0
	)

	now := time.Now().UTC()
	startDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	totDays := 30 * 12
	for day := 0; day < totDays; day++ {
		currentDate := startDate.AddDate(0, 0, day)

		// Skip Sundays (weekday 0)
		if currentDate.Weekday() == time.Sunday {
			continue
		}
		for hour := 7; hour <= 21; hour++ {
			slotTime := time.Date(currentDate.Year(), currentDate.Month(), currentDate.Day(),
				hour, 0, 0, 0, time.UTC)

			_, err = db.Exec(`
				INSERT INTO slots (starts_at, people_count, disabled)
				VALUES ($1, $2, $3)
				ON CONFLICT (starts_at) DO NOTHING
			`, slotTime, 0, false)
			if err != nil {
				log.Printf("Warning: Could not create slot at %v: %v", slotTime, err)
			} else {
				slotsCreated++
			}
		}
	}
	log.Printf("created %d slots", slotsCreated)
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
