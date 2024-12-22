package main

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"flag"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	_ "github.com/lib/pq"
)

type Question struct {
	Id       int
	Sku      string
	Index    int
	Previous int
	Next     int
	Question string
}

const (
	createTable = `CREATE TABLE IF NOT EXISTS questions (
		id SERIAL PRIMARY KEY,
		sku VARCHAR(255) UNIQUE NOT NULL,
		index INTEGER NOT NULL,
		next INTEGER NOT NULL,
		previous INTEGER NOT NULL,
		question TEXT NOT NULL,
		star1 INTEGER NOT NULL DEFAULT 0, 
		star2 INTEGER NOT NULL DEFAULT 0, 
		star3 INTEGER NOT NULL DEFAULT 0, 
		star4 INTEGER NOT NULL DEFAULT 0, 
		star5 INTEGER NOT NULL DEFAULT 0
	)`
)

var (
	//go:embed templates static
	files        embed.FS
	listenAddr   = flag.String("listen-addr", "localhost:3000", "Listen address for the web application")
	dbConnString = flag.String("db-uri", "postgresql://postgres:postgres@localhost:5432/postgres?sslmode=disable", "Database connection string")
	tpl          *template.Template
)

func init() {
	tpl = template.Must(template.ParseFS(files, "templates/*.html"))
}

func main() {
	flag.Parse()

	var (
		ctx = context.Background()
	)

	content, err := fs.Sub(files, "static")
	if err != nil {
		log.Fatal(err)
	}
	db, err := sql.Open("postgres", *dbConnString)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if _, err := db.Exec(createTable); err != nil {
		log.Fatal(err)
	}
	ctx, canc := signal.NotifyContext(ctx, syscall.SIGTERM, os.Interrupt)
	defer canc()

	if err := run(ctx, db, *listenAddr, content); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, db *sql.DB, listenAddres string, staticContent fs.FS) error {

	fs := http.FileServer(http.FS(staticContent))
	http.HandleFunc("/static/", func(wr http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet {
			http.Error(wr, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		ext := filepath.Ext(req.URL.Path)
		// Determine mime type based on the URL
		if ext == ".css" {
			wr.Header().Set("Content-Type", "text/css")
		} else if ext == ".js" {
			wr.Header().Set("Content-Type", "text/javascript")
		}
		http.StripPrefix("/static/", fs).ServeHTTP(wr, req)
	})

	http.HandleFunc("/submit", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			log.Print(err)
			return
		}
		if err := updateResults(db, r.Form); err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			log.Print(err)
			return
		}
		http.Redirect(w, r, "/thanks.html", http.StatusMovedPermanently)
	})

	http.HandleFunc("/", serveTemplate(db))

	log.Printf("listening on %s", listenAddres)
	return startHttpServer(ctx, http.DefaultServeMux, listenAddres)
}

func serveTemplate(db *sql.DB) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		if r.URL.Path == "/" {
			r.URL.Path = "/index.html"
		}
		questions, err := loadQuestions(db)

		if err != nil {
			log.Print(err.Error())
			http.Error(w, http.StatusText(500), 500)
			return
		}
		if err = tpl.ExecuteTemplate(w, r.URL.Path, questions); err != nil {
			log.Print(err.Error())
			http.Error(w, http.StatusText(500), 500)
		}

	})
}
func startHttpServer(ctx context.Context, r *http.ServeMux, addr string) error {
	server := http.Server{
		Addr:              addr,
		Handler:           r,
		ReadTimeout:       time.Minute,
		WriteTimeout:      time.Minute,
		IdleTimeout:       time.Minute,
		ReadHeaderTimeout: 10 * time.Second,
		MaxHeaderBytes:    1024 * 8,
	}

	errCh := make(chan error)
	defer close(errCh)
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
	case err := <-errCh:
		return err
	}

	ctx, canc := context.WithTimeout(context.Background(), time.Second*10)
	defer canc()
	return server.Shutdown(ctx)
}

func loadQuestions(db *sql.DB) ([]Question, error) {
	stmt := "SELECT id, sku, index, next, previous, question FROM questions ORDER BY index"
	rows, err := db.Query(stmt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var questions []Question
	for rows.Next() {
		var question Question
		err := rows.Scan(
			&question.Id,
			&question.Sku,
			&question.Index,
			&question.Next,
			&question.Previous,
			&question.Question)
		if err != nil {
			return questions, err
		}
		questions = append(questions, question)
	}

	return questions, nil

}

func updateResults(db *sql.DB, results url.Values) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	stmt := `UPDATE questions SET
	 star1 = star1 + $1,
	 star2 = star2 + $2,
	 star3 = star3 + $3,
	 star4 = star4 + $4,
	 star5 = star5 + $5
	 WHERE id = $6
	`
	scores := make(map[int][]int, len(results))
	for k, v := range results {
		id, err := strconv.Atoi(strings.Split(k, "-")[1])
		if err != nil {
			return err
		}
		scores[id] = make([]int, 5)
		star, err := strconv.Atoi(v[0])
		if err != nil {
			return err
		}
		scores[id][star-1] = 1
		if _, err := db.Exec(stmt, scores[id][0], scores[id][1], scores[id][2], scores[id][3], scores[id][4], id); err != nil {
			return err
		}

	}

	return tx.Commit()
}
