package main

import (
	"database/sql"
	"log"
	"os"
	"time"

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

	if err := clearSlots(db); err != nil {
		log.Print(err)
	}
	if err := clearEvents(db); err != nil {
		log.Print(err)
	}

	log.Printf("cleanup executed on %s", time.Now())
}

func clearSlots(db *sql.DB) error {
	query := "DELETE FROM \"Slot\" where \"startsAt\" < now() - interval '3 months'"
	_, err := db.Exec(query)
	return err
}

func clearEvents(db *sql.DB) error {
	query := "DELETE FROM \"Event\" where \"startsAt\" < now() - interval '1 months'"
	_, err := db.Exec(query)
	return err
}
