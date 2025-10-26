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
	// emailNotify := os.Getenv("EMAIL_NOTIFY_ADDRESS")
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
	_ = handlers.NewPageHandler(userRepo, bookingRepo, eventRepo) // Page handler logic moved to main.go serve functions

	mux := http.NewServeMux()

	// Static files
	fs := http.FileServer(http.FS(staticContent))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))

	// Public routes
	mux.HandleFunc("GET /signin", serveSignIn)
	mux.HandleFunc("GET /reset", serveReset)
	mux.HandleFunc("GET /verify", serveVerify)
	mux.HandleFunc("GET /survey", serveSurvey(questionRepo))
	mux.HandleFunc("GET /survey/thanks", serveSurveyThanks)

	// Auth API routes
	mux.HandleFunc("POST /api/auth/login", authHandler.Login)
	mux.HandleFunc("GET  /api/auth/logout", authHandler.Logout)
	mux.HandleFunc("POST /api/auth/reset", userHandler.ResetPassword)
	mux.HandleFunc("POST /api/auth/verify", userHandler.VerifyAccount)

	// Public survey API routes
	mux.HandleFunc("POST /survey/submit", surveyHandler.SubmitSurvey)

	// User dashboard
	authMiddleware := middleware.Auth(sessionStore, userRepo)
	mux.Handle("GET /user", authMiddleware(http.HandlerFunc(serveUserDashboard(bookingRepo, instructorRepo))))
	mux.Handle("GET /user/", authMiddleware(http.HandlerFunc(serveUserDashboard(bookingRepo, instructorRepo))))

	// User API
	mux.Handle("GET /api/user/bookings", authMiddleware(http.HandlerFunc(bookingHandler.GetCurrent)))
	mux.Handle("POST /api/user/bookings", authMiddleware(http.HandlerFunc(bookingHandler.Create)))
	mux.Handle("DELETE /api/user/bookings/{id}", authMiddleware(http.HandlerFunc(bookingHandler.Delete)))
	mux.Handle("GET /api/user/bookings/slots", authMiddleware(http.HandlerFunc(bookingHandler.GetAvailableSlots)))

	// Admin dashboard
	adminMiddleware := middleware.AdminAuth(sessionStore, userRepo)
	mux.Handle("GET /admin", authMiddleware(http.HandlerFunc(serveAdminHome)))
	mux.Handle("GET /admin/", authMiddleware(http.HandlerFunc(serveAdminHome)))
	mux.Handle("GET /admin/calendar", adminMiddleware(http.HandlerFunc(serveCalendar)))
	mux.Handle("GET /admin/users", adminMiddleware(http.HandlerFunc(serveUsers(userRepo))))
	mux.Handle("GET /admin/instructors", adminMiddleware(http.HandlerFunc(serveInstructors(instructorRepo))))
	mux.Handle("GET /admin/events", adminMiddleware(http.HandlerFunc(serveEvents(userRepo, eventRepo))))
	mux.Handle("GET /admin/survey/questions", adminMiddleware(http.HandlerFunc(serveSurveyQuestions)))
	mux.Handle("GET /admin/survey/results", adminMiddleware(http.HandlerFunc(serveSurveyResults)))

	// Admin user API
	mux.Handle("GET /api/admin/users", adminMiddleware(http.HandlerFunc(userHandler.GetAll)))
	mux.Handle("POST /api/admin/users", adminMiddleware(http.HandlerFunc(userHandler.Create)))
	mux.Handle("PUT /api/admin/users", adminMiddleware(http.HandlerFunc(userHandler.Update)))
	mux.Handle("DELETE /api/admin/users", adminMiddleware(http.HandlerFunc(userHandler.Delete)))
	mux.Handle("POST /api/admin/users/resend-verification", adminMiddleware(http.HandlerFunc(userHandler.ResendVerification)))

	// Instructors API - GetAll is accessible to all authenticated users (users need to see instructors for booking)
	mux.Handle("GET /api/user/instructors", authMiddleware(http.HandlerFunc(instructorHandler.GetAll)))
	mux.Handle("GET /api/admin/instructors", adminMiddleware(http.HandlerFunc(instructorHandler.GetAll)))
	mux.Handle("POST /api/admin/instructors", adminMiddleware(http.HandlerFunc(instructorHandler.Create)))
	mux.Handle("PUT /api/admin/instructors/{id}", adminMiddleware(http.HandlerFunc(instructorHandler.Update)))
	mux.Handle("DELETE /api/admin/instructors/{id}", adminMiddleware(http.HandlerFunc(instructorHandler.Delete)))

	// Bookings API
	mux.Handle("GET /api/admin/bookings", adminMiddleware(http.HandlerFunc(bookingHandler.GetAllBookings)))
	mux.Handle("POST /api/admin/bookings", adminMiddleware(http.HandlerFunc(bookingHandler.CreateBookingForUser)))
	mux.Handle("DELETE /api/admin/bookings/{id}", adminMiddleware(http.HandlerFunc(bookingHandler.DeleteAdmin)))

	// Survey API
	mux.Handle("GET /api/admin/survey/questions", adminMiddleware(http.HandlerFunc(surveyHandler.GetAllQuestions)))
	mux.Handle("POST /api/admin/survey/questions", adminMiddleware(http.HandlerFunc(surveyHandler.CreateQuestion)))
	mux.Handle("PUT /api/admin/survey/questions", adminMiddleware(http.HandlerFunc(surveyHandler.UpdateQuestion)))
	mux.Handle("DELETE /api/admin/survey/questions", adminMiddleware(http.HandlerFunc(surveyHandler.DeleteQuestion)))
	mux.Handle("GET /api/admin/survey/results", adminMiddleware(http.HandlerFunc(surveyHandler.GetResults)))

	// WebSocket endpoint
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		websocket.ServeWs(hub, w, r)
	})

	// Root redirect based on role
	mux.Handle("/", authMiddleware(http.HandlerFunc(serveRoot())))

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

func serveRoot() http.HandlerFunc {
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
		if user != nil && user.Role == models.RoleAdmin {
			http.Redirect(w, r, "/admin/calendar", http.StatusSeeOther)
		} else {
			http.Redirect(w, r, "/user", http.StatusSeeOther)
		}
	}
}

func serveUserDashboard(bookingRepo *models.BookingRepository, instructorRepo *models.InstructorRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := middleware.GetUserFromContext(r.Context())
		if user == nil {
			http.Redirect(w, r, "/signin", http.StatusSeeOther)
			return
		}

		// Only regular users can access user dashboard
		if user != nil && user.Role == models.RoleAdmin {
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
			ID                int64
			StartsAt          string
			StartsAtFormatted string
			CreatedAt         string
			InstructorName    string
		}

		var displayBookings []BookingDisplay
		for _, b := range bookings {
			instructorName := ""
			instructor, err := instructorRepo.GetByID(b.InstructorID)
			if err == nil {
				instructorName = instructor.FirstName
				if instructor.LastName != "" {
					instructorName += " " + instructor.LastName
				}
			}

			displayBookings = append(displayBookings, BookingDisplay{
				ID:                b.ID,
				StartsAt:          b.StartsAt.Format(time.RFC3339),
				StartsAtFormatted: b.StartsAt.Format("02 Jan 2006, 15:04"),
				CreatedAt:         b.CreatedAt.Format(time.RFC3339),
				InstructorName:    instructorName,
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
	if user == nil || user == nil || user.Role != models.RoleAdmin {
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
	if user == nil || user == nil || user.Role != models.RoleAdmin {
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
	data := map[string]interface{}{
		"Error": r.URL.Query().Get("error"),
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tpl.ExecuteTemplate(w, "signin.html", data); err != nil {
		log.Print(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func serveReset(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Token":   r.URL.Query().Get("token"),
		"Success": r.URL.Query().Get("success"),
		"Error":   r.URL.Query().Get("error"),
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tpl.ExecuteTemplate(w, "reset.html", data); err != nil {
		log.Print(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func serveVerify(w http.ResponseWriter, r *http.Request) {
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
}

func serveUsers(userRepo *models.UserRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		users, err := userRepo.GetAll()
		if err != nil {
			log.Printf("Error getting users: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// Format user data for display
		type UserDisplay struct {
			ID                 string
			FirstName          string
			LastName           string
			Email              string
			Address            string
			Cellphone          string
			SubType            string
			MedOk              bool
			EmailVerified      bool
			ExpiresAt          string
			ExpiresAtFormatted string
			RemainingAccesses  int
			Goals              string
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
				ID:                 u.ID,
				FirstName:          u.FirstName,
				LastName:           u.LastName,
				Email:              u.Email,
				Address:            u.Address,
				Cellphone:          cellphone,
				SubType:            string(u.SubType),
				MedOk:              u.MedOk,
				EmailVerified:      u.EmailVerified.Valid,
				ExpiresAt:          u.ExpiresAt.Format("2006-01-02"),
				ExpiresAtFormatted: u.ExpiresAt.Format("02 Jan 2006"),
				RemainingAccesses:  u.RemainingAccesses,
				Goals:              goals,
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
		if user == nil || user == nil || user.Role != models.RoleAdmin {
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
			ID         int
			UserName   string
			Type       string
			OccurredAt string
			StartsAt   string
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
				ID:         e.ID,
				UserName:   userName,
				Type:       string(e.Type),
				OccurredAt: e.OccurredAt.Format(time.RFC3339),
				StartsAt:   e.StartsAt.Format(time.RFC3339),
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

func serveSurvey(questionRepo *models.QuestionRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		questions, err := questionRepo.GetAll()
		if err != nil {
			log.Printf("Error getting questions: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		data := map[string]interface{}{
			"Questions": questions,
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := tpl.ExecuteTemplate(w, "survey.html", data); err != nil {
			log.Print(err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	}
}

func serveSurveyThanks(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tpl.ExecuteTemplate(w, "survey-thanks.html", nil); err != nil {
		log.Print(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func serveSurveyQuestions(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil || user == nil || user.Role != models.RoleAdmin {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tpl.ExecuteTemplate(w, "survey-questions.html", nil); err != nil {
		log.Print(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func serveSurveyResults(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	user := middleware.GetUserFromContext(r.Context())
	if user == nil || user == nil || user.Role != models.RoleAdmin {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tpl.ExecuteTemplate(w, "survey-results.html", nil); err != nil {
		log.Print(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func serveInstructors(instructorRepo *models.InstructorRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		user := middleware.GetUserFromContext(r.Context())
		if user == nil || user == nil || user.Role != models.RoleAdmin {
			http.Redirect(w, r, "/signin", http.StatusSeeOther)
			return
		}

		instructors, err := instructorRepo.GetAll()
		if err != nil {
			log.Printf("Error getting instructors: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// Format instructor data for display
		type InstructorDisplay struct {
			ID        int64
			FirstName string
			LastName  string
			CreatedAt string
		}

		var displayInstructors []InstructorDisplay
		for _, i := range instructors {
			displayInstructors = append(displayInstructors, InstructorDisplay{
				ID:        i.ID,
				FirstName: i.FirstName,
				LastName:  i.LastName,
				CreatedAt: i.CreatedAt.Format("02 Jan 2006"),
			})
		}

		data := map[string]interface{}{
			"Instructors": displayInstructors,
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := tpl.ExecuteTemplate(w, "instructors.html", data); err != nil {
			log.Print(err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	}
}
