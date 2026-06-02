package main

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alarmfox/wellness-nutrition/app/crypto"
	"github.com/alarmfox/wellness-nutrition/app/handlers"
	"github.com/alarmfox/wellness-nutrition/app/mail"
	"github.com/alarmfox/wellness-nutrition/app/middleware"
	"github.com/alarmfox/wellness-nutrition/app/models"
	"github.com/alarmfox/wellness-nutrition/app/websocket"
	_ "github.com/joho/godotenv/autoload"
	_ "github.com/lib/pq"
)

var (
	//go:embed templates static
	files embed.FS
	tpl   *template.Template
)

func init() {
	var err error
	tpl, err = template.ParseFS(files, "templates/*.html")
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	dbConnString := os.Getenv("DATABASE_URL")
	if dbConnString == "" {
		log.Fatal("database connection string is required")
	}

	listenAddr := os.Getenv("LISTEN_ADDR")
	if listenAddr == "" {
		log.Fatal("listenAddr is required")
	}

	// Initialize secret key for signing tokens and cookies
	secretKey := os.Getenv("SECRET_KEY")
	if secretKey == "" {
		log.Fatal("SECRET_KEY environment variable is required")
	}
	if err := crypto.InitializeSecretKey(secretKey); err != nil {
		log.Fatalf("failed to initialize secret key: %v", err)
	}

	ctx := context.Background()

	content, err := fs.Sub(files, "static")
	if err != nil {
		log.Fatal(err)
	}

	db, err := sql.Open("postgres", dbConnString)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}

	ctx, cancel := signal.NotifyContext(ctx, syscall.SIGTERM, os.Interrupt)
	defer cancel()

	if err := run(ctx, db, listenAddr, content); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, db *sql.DB, listenAddr string, staticContent fs.FS) error {
	// Initialize repositories
	userRepo := models.NewUserRepository(db)
	bookingRepo := models.NewBookingRepository(db)
	eventRepo := models.NewEventRepository(db)
	questionRepo := models.NewQuestionRepository(db)
	instructorRepo := models.NewInstructorRepository(db)

	// Initialize session store
	sessionStore := models.NewSessionStore(db)

	// Initialize mailer
	emailHost := os.Getenv("EMAIL_SERVER_HOST")
	emailPort := os.Getenv("EMAIL_SERVER_PORT")
	emailUser := os.Getenv("EMAIL_SERVER_USER")
	emailPassword := os.Getenv("EMAIL_SERVER_PASSWORD")
	emailFrom := os.Getenv("EMAIL_SERVER_FROM")
	mailer, err := mail.NewMailer(emailHost, emailPort, emailUser, emailPassword, emailFrom)
	if err != nil {
		return fmt.Errorf("failed to initialize mailer: %w", err)
	}

	// Start mailer goroutine
	go mailer.Run(ctx)

	// Initialize WebSocket hub
	hub := websocket.NewHub()
	go hub.Run(ctx)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(userRepo, sessionStore)
	userHandler := handlers.NewUserHandler(userRepo, mailer)
	bookingHandler := handlers.NewBookingHandler(bookingRepo, eventRepo, userRepo, instructorRepo, mailer, hub)
	instructorHandler := handlers.NewInstructorHandler(instructorRepo)
	surveyHandler := handlers.NewSurveyHandler(questionRepo)
	pageHandler := handlers.NewPageHandler(userRepo, bookingRepo, eventRepo, instructorRepo, questionRepo, tpl)

	mux := http.NewServeMux()

	// Static files - no CSRF needed
	fileServer := http.FileServer(http.FS(staticContent))
	mux.Handle("/static/", http.StripPrefix("/static/", fileServer))

	// CSRF middleware for protected routes
	csrfMiddleware := middleware.CSRF

	// Public routes - apply CSRF to set tokens in cookies for forms
	mux.Handle("GET /signin", csrfMiddleware(http.HandlerFunc(pageHandler.ServeSignIn)))
	mux.Handle("GET /reset", csrfMiddleware(http.HandlerFunc(pageHandler.ServeReset)))
	mux.Handle("GET /verify", csrfMiddleware(http.HandlerFunc(pageHandler.ServeVerify)))
	mux.Handle("GET /survey", csrfMiddleware(http.HandlerFunc(pageHandler.ServeSurvey)))
	mux.HandleFunc("GET /survey/thanks", pageHandler.ServeSurveyThanks)

	// Auth API routes - apply CSRF
	mux.Handle("POST /api/auth/login", csrfMiddleware(http.HandlerFunc(authHandler.Login)))
	mux.HandleFunc("DELETE /api/auth/logout", authHandler.Logout)
	mux.Handle("POST /api/auth/reset", csrfMiddleware(http.HandlerFunc(userHandler.ResetPassword)))
	mux.Handle("POST /api/auth/verify", csrfMiddleware(http.HandlerFunc(userHandler.VerifyAccount)))

	// Public survey API routes
	mux.Handle("POST /survey/submit", csrfMiddleware(http.HandlerFunc(surveyHandler.SubmitSurvey)))

	// User dashboard - apply CSRF
	authMiddleware := middleware.Auth(sessionStore, userRepo)
	mux.Handle("GET /user", csrfMiddleware(authMiddleware(http.HandlerFunc(pageHandler.ServeUserDashboard))))
	mux.Handle("GET /user/", csrfMiddleware(authMiddleware(http.HandlerFunc(pageHandler.ServeUserDashboard))))

	// User API - apply CSRF
	mux.Handle("GET /api/user/bookings", csrfMiddleware(authMiddleware(http.HandlerFunc(bookingHandler.GetCurrent))))
	mux.Handle("POST /api/user/bookings", csrfMiddleware(authMiddleware(http.HandlerFunc(bookingHandler.Create))))
	mux.Handle("DELETE /api/user/bookings/{id}", csrfMiddleware(authMiddleware(http.HandlerFunc(bookingHandler.Delete))))
	mux.Handle("GET /api/user/bookings/slots", csrfMiddleware(authMiddleware(http.HandlerFunc(bookingHandler.GetAvailableSlots))))

	// Admin dashboard - apply CSRF
	adminMiddleware := middleware.AdminAuth(sessionStore, userRepo)
	mux.Handle("GET /admin", csrfMiddleware(authMiddleware(http.HandlerFunc(pageHandler.ServeAdminHome))))
	mux.Handle("GET /admin/", csrfMiddleware(authMiddleware(http.HandlerFunc(pageHandler.ServeAdminHome))))
	mux.Handle("GET /admin/calendar", csrfMiddleware(adminMiddleware(http.HandlerFunc(pageHandler.ServeCalendar))))
	mux.Handle("GET /admin/users", csrfMiddleware(adminMiddleware(http.HandlerFunc(pageHandler.ServeUsers))))
	mux.Handle("GET /admin/instructors", csrfMiddleware(adminMiddleware(http.HandlerFunc(pageHandler.ServeInstructors))))
	mux.Handle("GET /admin/events", csrfMiddleware(adminMiddleware(http.HandlerFunc(pageHandler.ServeEvents))))
	mux.Handle("GET /admin/survey/questions", csrfMiddleware(adminMiddleware(http.HandlerFunc(pageHandler.ServeSurveyQuestions))))
	mux.Handle("GET /admin/survey/results", csrfMiddleware(adminMiddleware(http.HandlerFunc(pageHandler.ServeSurveyResults))))
	mux.Handle("GET /admin/user-view", csrfMiddleware(adminMiddleware(http.HandlerFunc(pageHandler.ServeUserView))))

	// Admin user API - apply CSRF
	mux.Handle("GET /api/admin/users", csrfMiddleware(adminMiddleware(http.HandlerFunc(userHandler.GetAll))))
	mux.Handle("POST /api/admin/users", csrfMiddleware(adminMiddleware(http.HandlerFunc(userHandler.Create))))
	mux.Handle("PUT /api/admin/users", csrfMiddleware(adminMiddleware(http.HandlerFunc(userHandler.Update))))
	mux.Handle("DELETE /api/admin/users", csrfMiddleware(adminMiddleware(http.HandlerFunc(userHandler.Delete))))
	mux.Handle("POST /api/admin/users/resend-verification", csrfMiddleware(adminMiddleware(http.HandlerFunc(userHandler.ResendVerification))))

	// Instructors API - apply CSRF
	mux.Handle("GET /api/user/instructors", csrfMiddleware(authMiddleware(http.HandlerFunc(instructorHandler.GetAll))))
	mux.Handle("GET /api/admin/instructors", csrfMiddleware(adminMiddleware(http.HandlerFunc(instructorHandler.GetAll))))
	mux.Handle("POST /api/admin/instructors", csrfMiddleware(adminMiddleware(http.HandlerFunc(instructorHandler.Create))))
	mux.Handle("PUT /api/admin/instructors/{id}", csrfMiddleware(adminMiddleware(http.HandlerFunc(instructorHandler.Update))))
	mux.Handle("DELETE /api/admin/instructors/{id}", csrfMiddleware(adminMiddleware(http.HandlerFunc(instructorHandler.Delete))))

	// Bookings API - apply CSRF
	mux.Handle("GET /api/admin/bookings", csrfMiddleware(adminMiddleware(http.HandlerFunc(bookingHandler.GetAllBookings))))
	mux.Handle("POST /api/admin/bookings", csrfMiddleware(adminMiddleware(http.HandlerFunc(bookingHandler.CreateBookingForUser))))
	mux.Handle("DELETE /api/admin/bookings/{id}", csrfMiddleware(adminMiddleware(http.HandlerFunc(bookingHandler.DeleteAdmin))))

	// Survey API - apply CSRF
	mux.Handle("GET /api/admin/survey/questions", csrfMiddleware(adminMiddleware(http.HandlerFunc(surveyHandler.GetAllQuestions))))
	mux.Handle("POST /api/admin/survey/questions", csrfMiddleware(adminMiddleware(http.HandlerFunc(surveyHandler.CreateQuestion))))
	mux.Handle("PUT /api/admin/survey/questions", csrfMiddleware(adminMiddleware(http.HandlerFunc(surveyHandler.UpdateQuestion))))
	mux.Handle("DELETE /api/admin/survey/questions", csrfMiddleware(adminMiddleware(http.HandlerFunc(surveyHandler.DeleteQuestion))))
	mux.Handle("GET /api/admin/survey/results", csrfMiddleware(adminMiddleware(http.HandlerFunc(surveyHandler.GetResults))))

	// WebSocket endpoint - no CSRF for websocket
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		websocket.ServeWs(hub, w, r)
	})

	// Root redirect based on role - apply CSRF
	mux.Handle("/", csrfMiddleware(authMiddleware(http.HandlerFunc(pageHandler.ServeRoot))))

	log.Printf("listening on %s", listenAddr)
	return startHTTPServer(ctx, mux, listenAddr)
}

func startHTTPServer(ctx context.Context, r *http.ServeMux, addr string) error {
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
