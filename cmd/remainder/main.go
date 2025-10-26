package main

import (
	"context"
	"database/sql"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alarmfox/wellness-nutrition/app/mail"
	_ "github.com/lib/pq"
)

func main() {
	databaseUrl := os.Getenv("DATABASE_URL")
	if databaseUrl == "" {
		log.Fatal("DATABASE_URL is missing")
	}
	db, err := sql.Open("postgres", databaseUrl)
	if err != nil {
		log.Fatal(err)
	}
	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}

	db.SetMaxIdleConns(0)
	db.SetMaxOpenConns(1)
	defer db.Close()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, os.Interrupt)
	defer cancel()

	emailHost := os.Getenv("EMAIL_SERVER_HOST")
	emailPort := os.Getenv("EMAIL_SERVER_PORT")
	emailUser := os.Getenv("EMAIL_SERVER_USER")
	emailPassword := os.Getenv("EMAIL_SERVER_PASSWORD")
	emailFrom := os.Getenv("EMAIL_SERVER_FROM")
	// emailNotify := os.Getenv("EMAIL_NOTIFY_ADDRESS")
	mailer, err := mail.NewMailer(emailHost, emailPort, emailUser, emailPassword, emailFrom)
	if err != nil {
		log.Fatal("failed to initialize mailer: %w", err)
	}

	go mailer.Run(ctx)

	type Booking struct {
		FirstName string
		Email     string
		StartsAt  time.Time
	}

	query := `
	SELECT u.first_name, u.email, b.starts_at
	FROM bookings b
	LEFT JOIN users u ON b.user_id =  u.id
	WHERE b.starts_at >= CURRENT_DATE
	AND b.starts_at < CURRENT_DATE + INTERVAL '1 day'
	AND b.type = 'SIMPLE'
	`

	rows, err := db.Query(query)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	var bookings []*Booking
	for rows.Next() {
		var booking Booking
		err := rows.Scan(
			&booking.FirstName,
			&booking.Email,
			&booking.StartsAt,
		)
		if err != nil {
			log.Fatal(err)
		}
		bookings = append(bookings, &booking)
	}

	for _, booking := range bookings {
		log.Printf("Sending notification to %s", booking.Email)
		if err := mailer.SendReminderEmail(booking.Email, booking.FirstName, booking.StartsAt); err != nil {
			log.Print(err)
		}
	}
}
