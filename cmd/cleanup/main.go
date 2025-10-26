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

	cleanupQueries := []string{
		"DELETE FROM slots WHERE starts_at < now() - interval '3 months'",
		"DELETE FROM events WHERE starts_at < now() - interval '1 months'",
	}

	for _, query := range cleanupQueries {
		_, err := db.Exec(query)
		if err != nil {
			log.Print(err)
		}
	}

	log.Printf("cleanup executed on %s", time.Now())
}
