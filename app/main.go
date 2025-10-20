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
	_ = handlers.NewPageHandler(userRepo, bookingRepo, eventRepo) // Page handler logic moved to main.go serve functions
	
	mux := http.NewServeMux()

	// Static files
	fs := http.FileServer(http.FS(staticContent))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))

	// Public routes
	mux.HandleFunc("/signin", serveSignIn)
	mux.HandleFunc("/reset", serveReset)
	mux.HandleFunc("/verify", serveVerify)
	
	// Auth API routes
	mux.HandleFunc("/api/auth/login", authHandler.Login)
	mux.HandleFunc("/api/auth/logout", authHandler.Logout)
	mux.HandleFunc("/api/auth/reset", userHandler.ResetPassword)
	mux.HandleFunc("/api/auth/verify", userHandler.VerifyAccount)
	
	// User routes - /user prefix
	authMiddleware := middleware.Auth(sessionStore, userRepo)
	mux.Handle("/user", authMiddleware(http.HandlerFunc(serveUserDashboard(db, bookingRepo))))
	mux.Handle("/user/", authMiddleware(http.HandlerFunc(serveUserDashboard(db, bookingRepo))))
	mux.Handle("/api/user/current", authMiddleware(http.HandlerFunc(userHandler.GetCurrent)))
	mux.Handle("/api/user/bookings", authMiddleware(http.HandlerFunc(bookingHandler.GetCurrent)))
	mux.Handle("/api/user/bookings/create", authMiddleware(http.HandlerFunc(bookingHandler.Create)))
	mux.Handle("/api/user/bookings/delete", authMiddleware(http.HandlerFunc(bookingHandler.Delete)))
	mux.Handle("/api/user/bookings/slots", authMiddleware(http.HandlerFunc(bookingHandler.GetAvailableSlots)))
	
	// Admin routes - /admin prefix
	adminMiddleware := middleware.AdminAuth(sessionStore, userRepo)
	mux.Handle("/admin", authMiddleware(http.HandlerFunc(serveAdminHome)))
	mux.Handle("/admin/", authMiddleware(http.HandlerFunc(serveAdminHome)))
	mux.Handle("/admin/calendar", adminMiddleware(http.HandlerFunc(serveCalendar)))
	mux.Handle("/admin/users", adminMiddleware(http.HandlerFunc(serveUsers(userRepo))))
	mux.Handle("/admin/events", adminMiddleware(http.HandlerFunc(serveEvents(userRepo, eventRepo))))
	mux.Handle("/api/admin/users", adminMiddleware(http.HandlerFunc(userHandler.GetAll)))
	mux.Handle("/api/admin/users/create", adminMiddleware(http.HandlerFunc(userHandler.Create)))
	mux.Handle("/api/admin/users/update", adminMiddleware(http.HandlerFunc(userHandler.Update)))
	mux.Handle("/api/admin/users/delete", adminMiddleware(http.HandlerFunc(userHandler.Delete)))
	mux.Handle("/api/admin/users/resend-verification", adminMiddleware(http.HandlerFunc(userHandler.ResendVerification)))
	mux.Handle("/api/admin/bookings", adminMiddleware(http.HandlerFunc(bookingHandler.GetAllBookings)))
	mux.Handle("/api/admin/bookings/create", adminMiddleware(http.HandlerFunc(bookingHandler.CreateBookingForUser)))
	mux.Handle("/api/admin/slots", adminMiddleware(http.HandlerFunc(bookingHandler.GetAllSlots)))
	mux.Handle("/api/admin/slots/disable", adminMiddleware(http.HandlerFunc(bookingHandler.DisableSlot)))
	mux.Handle("/api/admin/slots/disable-confirm", adminMiddleware(http.HandlerFunc(bookingHandler.DisableSlotConfirm)))
	mux.Handle("/api/admin/slots/enable", adminMiddleware(http.HandlerFunc(bookingHandler.EnableSlot)))
	
	// Root redirect based on role
	mux.Handle("/", authMiddleware(http.HandlerFunc(serveRoot(db, bookingRepo))))

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

func serveRoot(db *sql.DB, bookingRepo *models.BookingRepository) http.HandlerFunc {
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
		
		// Redirect based on role
		if user.Role == models.RoleAdmin {
			http.Redirect(w, r, "/admin/calendar", http.StatusSeeOther)
		} else {
			http.Redirect(w, r, "/user", http.StatusSeeOther)
		}
	}
}

func serveUserDashboard(db *sql.DB, bookingRepo *models.BookingRepository) http.HandlerFunc {
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
		
		// Only regular users can access user dashboard
		if user.Role == models.RoleAdmin {
			http.Redirect(w, r, "/admin/calendar", http.StatusSeeOther)
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

func serveAdminHome(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil || user.Role != models.RoleAdmin {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}
	
	// Redirect admin to calendar
	http.Redirect(w, r, "/admin/calendar", http.StatusSeeOther)
}

func serveCalendar(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	
	user := middleware.GetUserFromContext(r.Context())
	if user == nil || user.Role != models.RoleAdmin {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}
	
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tpl.ExecuteTemplate(w, "calendar.html", nil); err != nil {
		log.Print(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
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

func serveReset(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		data := map[string]interface{}{
			"Success": r.URL.Query().Get("success"),
			"Error":   r.URL.Query().Get("error"),
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := tpl.ExecuteTemplate(w, "reset.html", data); err != nil {
			log.Print(err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
}

func serveVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		token := r.URL.Query().Get("token")
		data := map[string]interface{}{
			"Token": token,
			"Error": r.URL.Query().Get("error"),
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := tpl.ExecuteTemplate(w, "verify.html", data); err != nil {
			log.Print(err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
}

func serveUsers(userRepo *models.UserRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		
		user := middleware.GetUserFromContext(r.Context())
		if user == nil || user.Role != models.RoleAdmin {
			http.Redirect(w, r, "/signin", http.StatusSeeOther)
			return
		}
		
		users, err := userRepo.GetAll()
		if err != nil {
			log.Printf("Error getting users: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		
		// Format user data for display
		type UserDisplay struct {
			ID                  string
			FirstName           string
			LastName            string
			Email               string
			Address             string
			Cellphone           string
			SubType             string
			MedOk               bool
			EmailVerified       bool
			ExpiresAt           string
			ExpiresAtFormatted  string
			RemainingAccesses   int
			Goals               string
		}
		
		var displayUsers []UserDisplay
		for _, u := range users {
			cellphone := ""
			if u.Cellphone.Valid {
				cellphone = u.Cellphone.String
			}
			goals := ""
			if u.Goals.Valid {
				goals = u.Goals.String
			}
			displayUsers = append(displayUsers, UserDisplay{
				ID:                  u.ID,
				FirstName:           u.FirstName,
				LastName:            u.LastName,
				Email:               u.Email,
				Address:             u.Address,
				Cellphone:           cellphone,
				SubType:             string(u.SubType),
				MedOk:               u.MedOk,
				EmailVerified:       u.EmailVerified.Valid,
				ExpiresAt:           u.ExpiresAt.Format("2006-01-02"),
				ExpiresAtFormatted:  u.ExpiresAt.Format("02 Jan 2006"),
				RemainingAccesses:   u.RemainingAccesses,
				Goals:               goals,
			})
		}
		
		data := map[string]interface{}{
			"Users": displayUsers,
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := tpl.ExecuteTemplate(w, "users.html", data); err != nil {
			log.Print(err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	}
}

func serveEvents(userRepo *models.UserRepository, eventRepo *models.EventRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		
		user := middleware.GetUserFromContext(r.Context())
		if user == nil || user.Role != models.RoleAdmin {
			http.Redirect(w, r, "/signin", http.StatusSeeOther)
			return
		}
		
		events, err := eventRepo.GetAll()
		if err != nil {
			log.Printf("Error getting events: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		
		// Format event data for display
		type EventDisplay struct {
			ID                  int
			UserName            string
			Type                string
			OccurredAtFormatted string
			StartsAtFormatted   string
		}
		
		var displayEvents []EventDisplay
		for _, e := range events {
			// Get user for event
			u, err := userRepo.GetByID(e.UserID)
			userName := "Unknown"
			if err == nil {
				userName = u.FirstName + " " + u.LastName
			}
			
			displayEvents = append(displayEvents, EventDisplay{
				ID:                  e.ID,
				UserName:            userName,
				Type:                string(e.Type),
				OccurredAtFormatted: e.OccurredAt.Format("02 Jan 2006 15:04"),
				StartsAtFormatted:   e.StartsAt.Format("02 Jan 2006 15:04"),
			})
		}
		
		data := map[string]interface{}{
			"Events": displayEvents,
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := tpl.ExecuteTemplate(w, "events.html", data); err != nil {
			log.Print(err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	}
}
