package main

import (
	"database/sql"
	"embed"
	"flag"
	"fmt"
	"log"
	"os"
	"sort"

	_ "github.com/lib/pq"
)

var (
	//go:embed *.sql
	migrations  embed.FS
	dbConnString = flag.String("db-uri", "", "Database connection string")
)

func main() {
	flag.Parse()

	if *dbConnString == "" {
		*dbConnString = os.Getenv("DATABASE_URL")
	}
	if *dbConnString == "" {
		log.Fatal("database connection string is required (use -db-uri flag or DATABASE_URL env var)")
	}

	db, err := sql.Open("postgres", *dbConnString)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}

	log.Println("Running migrations...")

	// Read all migration files
	entries, err := migrations.ReadDir(".")
	if err != nil {
		log.Fatal(err)
	}

	// Sort migration files
	var migrationFiles []string
	for _, entry := range entries {
		if !entry.IsDir() && len(entry.Name()) > 4 && entry.Name()[len(entry.Name())-4:] == ".sql" {
			migrationFiles = append(migrationFiles, entry.Name())
		}
	}
	sort.Strings(migrationFiles)

	// Execute each migration
	for _, filename := range migrationFiles {
		log.Printf("Applying migration: %s", filename)
		
		content, err := migrations.ReadFile(filename)
		if err != nil {
			log.Fatalf("Error reading %s: %v", filename, err)
		}

		_, err = db.Exec(string(content))
		if err != nil {
			log.Fatalf("Error executing %s: %v", filename, err)
		}

		log.Printf("✓ Migration %s applied successfully", filename)
	}

	log.Println("\n=== All migrations completed successfully ===")
	
	// Display table information
	log.Println("\nDatabase tables created:")
	rows, err := db.Query(`
		SELECT table_name 
		FROM information_schema.tables 
		WHERE table_schema = 'public' 
		ORDER BY table_name
	`)
	if err != nil {
		log.Printf("Warning: Could not query tables: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			continue
		}
		fmt.Printf("  - %s\n", tableName)
	}
}
