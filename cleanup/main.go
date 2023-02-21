package main

import (
	"database/sql"
	"flag"
	"log"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

var (
	databaseUrl = flag.String("database-url", "", "Url to database (POSTGRES ONLY)")
	cmd         = flag.String("cmd", "", "Command: slots (clear slots older than 3 months), events (clear events older than 1 month)")
)

func main() {
	flag.Parse()
	db, err := sql.Open("postgres", *databaseUrl)

	if err != nil {
		log.Fatal(err)
	}
  db.SetMaxIdleConns(0)
	db.SetMaxOpenConns(1)
	defer db.Close()
	
	switch strings.ToLower(*cmd) {
	case "slots":
		if err := clearSlots(db); err != nil {
			log.Print(err)
		}
	case "events":
		if err := clearEvents(db); err != nil {
			log.Print(err)
		}
	default:
		log.Printf("unknown command %q", *cmd)
		return
	}
	log.Printf("%s cmd executed on %s", *cmd, time.Now())
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
