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
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alarmfox/wellness-nutrition/app/handlers"
	"github.com/alarmfox/wellness-nutrition/app/mail"
	"github.com/alarmfox/wellness-nutrition/app/middleware"
	"github.com/alarmfox/wellness-nutrition/app/models"
	_ "github.com/lib/pq"
)

var (
	//go:embed templates static
	files        embed.FS
	listenAddr   = flag.String("listen-addr", "localhost:3000", "Listen address for the web application")
	dbConnString = flag.String("db-uri", "", "Database connection string")
	tpl          *template.Template
)

func init() {
	var err error
	tpl, err = template.ParseFS(files, "templates/*.html")
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	flag.Parse()

	if *dbConnString == "" {
		*dbConnString = os.Getenv("DATABASE_URL")
	}
	if *dbConnString == "" {
		log.Fatal("database connection string is required")
	}

	ctx := context.Background()

	content, err := fs.Sub(files, "static")
	if err != nil {
		log.Fatal(err)
	}

	db, err := sql.Open("postgres", *dbConnString)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}

	ctx, cancel := signal.NotifyContext(ctx, syscall.SIGTERM, os.Interrupt)
	defer cancel()

	if err := run(ctx, db, *listenAddr, content); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, db *sql.DB, listenAddr string, staticContent fs.FS) error {
	// Initialize repositories
	userRepo := models.NewUserRepository(db)
	bookingRepo := models.NewBookingRepository(db)
	slotRepo := models.NewSlotRepository(db)
	eventRepo := models.NewEventRepository(db)
	
	// Initialize session store
	sessionStore := middleware.NewSessionStore(db)
	if err := sessionStore.InitTable(); err != nil {
		log.Printf("Warning: Could not initialize session table: %v", err)
	}
	
	// Initialize mailer
	mailer := mail.NewMailer()
	
	// Initialize handlers
	authHandler := handlers.NewAuthHandler(userRepo, sessionStore)
	userHandler := handlers.NewUserHandler(userRepo, mailer)
	bookingHandler := handlers.NewBookingHandler(bookingRepo, slotRepo, eventRepo, userRepo, mailer)
	
	mux := http.NewServeMux()

	// Static files
	fs := http.FileServer(http.FS(staticContent))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))

	// Public routes
	mux.HandleFunc("/signin", serveSignIn)
	
	// Auth API routes
	mux.HandleFunc("/api/auth/login", authHandler.Login)
	mux.HandleFunc("/api/auth/logout", authHandler.Logout)
	
	// Protected routes - User
	authMiddleware := middleware.Auth(sessionStore, userRepo)
	mux.Handle("/", authMiddleware(http.HandlerFunc(serveHome(db, bookingRepo))))
	mux.Handle("/api/user/current", authMiddleware(http.HandlerFunc(userHandler.GetCurrent)))
	mux.Handle("/api/bookings/current", authMiddleware(http.HandlerFunc(bookingHandler.GetCurrent)))
	mux.Handle("/api/bookings/create", authMiddleware(http.HandlerFunc(bookingHandler.Create)))
	mux.Handle("/api/bookings/delete", authMiddleware(http.HandlerFunc(bookingHandler.Delete)))
	mux.Handle("/api/bookings/slots", authMiddleware(http.HandlerFunc(bookingHandler.GetAvailableSlots)))
	
	// Protected routes - Admin only
	adminMiddleware := middleware.AdminAuth(sessionStore, userRepo)
	mux.Handle("/api/admin/users", adminMiddleware(http.HandlerFunc(userHandler.GetAll)))
	mux.Handle("/api/admin/users/create", adminMiddleware(http.HandlerFunc(userHandler.Create)))

	log.Printf("listening on %s", listenAddr)
	return startHttpServer(ctx, mux, listenAddr)
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

	shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	return server.Shutdown(shutdownCtx)
}

func serveHome(db *sql.DB, bookingRepo *models.BookingRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		
		user := middleware.GetUserFromContext(r.Context())
		if user == nil {
			http.Redirect(w, r, "/signin", http.StatusSeeOther)
			return
		}

		// Get user's bookings
		bookings, err := bookingRepo.GetByUserID(user.ID)
		if err != nil {
			log.Printf("Error getting bookings: %v", err)
			bookings = []*models.Booking{}
		}
		
		// Format dates for display
		type BookingDisplay struct {
			ID                 int64
			StartsAt           string
			StartsAtFormatted  string
			CreatedAtFormatted string
		}
		
		var displayBookings []BookingDisplay
		for _, b := range bookings {
			displayBookings = append(displayBookings, BookingDisplay{
				ID:                 b.ID,
				StartsAt:           b.StartsAt.Format(time.RFC3339),
				StartsAtFormatted:  b.StartsAt.Format("02 Jan 2006 15:04"),
				CreatedAtFormatted: b.CreatedAt.Format("02 Jan 2006"),
			})
		}
		
		data := map[string]interface{}{
			"User":              user,
			"ExpiresAt":         user.ExpiresAt.Format("02 Jan 2006"),
			"RemainingAccesses": user.RemainingAccesses,
			"Bookings":          displayBookings,
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := tpl.ExecuteTemplate(w, "index.html", data); err != nil {
			log.Print(err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	}
}

func serveSignIn(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		data := map[string]interface{}{
			"Error": r.URL.Query().Get("error"),
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := tpl.ExecuteTemplate(w, "signin.html", data); err != nil {
			log.Print(err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
}
