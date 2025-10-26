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

	// Static files - no CSRF needed
	fs := http.FileServer(http.FS(staticContent))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))

	// CSRF middleware for protected routes
	csrfMiddleware := middleware.CSRF

	// Public routes - apply CSRF to set tokens in cookies for forms
	mux.Handle("GET /signin", csrfMiddleware(http.HandlerFunc(serveSignIn)))
	mux.Handle("GET /reset", csrfMiddleware(http.HandlerFunc(serveReset)))
	mux.Handle("GET /verify", csrfMiddleware(http.HandlerFunc(serveVerify)))
	mux.Handle("GET /survey", csrfMiddleware(http.HandlerFunc(serveSurvey(questionRepo))))
	mux.HandleFunc("GET /survey/thanks", serveSurveyThanks)

	// Auth API routes - apply CSRF
	mux.Handle("POST /api/auth/login", csrfMiddleware(http.HandlerFunc(authHandler.Login)))
	mux.HandleFunc("GET  /api/auth/logout", authHandler.Logout)
	mux.Handle("POST /api/auth/reset", csrfMiddleware(http.HandlerFunc(userHandler.ResetPassword)))
	mux.Handle("POST /api/auth/verify", csrfMiddleware(http.HandlerFunc(userHandler.VerifyAccount)))

	// Public survey API routes
	mux.Handle("POST /survey/submit", csrfMiddleware(http.HandlerFunc(surveyHandler.SubmitSurvey)))

	// User dashboard - apply CSRF
	authMiddleware := middleware.Auth(sessionStore, userRepo)
	mux.Handle("GET /user", csrfMiddleware(authMiddleware(http.HandlerFunc(serveUserDashboard(bookingRepo, instructorRepo)))))
	mux.Handle("GET /user/", csrfMiddleware(authMiddleware(http.HandlerFunc(serveUserDashboard(bookingRepo, instructorRepo)))))

	// User API - apply CSRF
	mux.Handle("GET /api/user/bookings", csrfMiddleware(authMiddleware(http.HandlerFunc(bookingHandler.GetCurrent))))
	mux.Handle("POST /api/user/bookings", csrfMiddleware(authMiddleware(http.HandlerFunc(bookingHandler.Create))))
	mux.Handle("DELETE /api/user/bookings/{id}", csrfMiddleware(authMiddleware(http.HandlerFunc(bookingHandler.Delete))))
	mux.Handle("GET /api/user/bookings/slots", csrfMiddleware(authMiddleware(http.HandlerFunc(bookingHandler.GetAvailableSlots))))

	// Admin dashboard - apply CSRF
	adminMiddleware := middleware.AdminAuth(sessionStore, userRepo)
	mux.Handle("GET /admin", csrfMiddleware(authMiddleware(http.HandlerFunc(serveAdminHome))))
	mux.Handle("GET /admin/", csrfMiddleware(authMiddleware(http.HandlerFunc(serveAdminHome))))
	mux.Handle("GET /admin/calendar", csrfMiddleware(adminMiddleware(http.HandlerFunc(serveCalendar))))
	mux.Handle("GET /admin/users", csrfMiddleware(adminMiddleware(http.HandlerFunc(serveUsers(userRepo)))))
	mux.Handle("GET /admin/instructors", csrfMiddleware(adminMiddleware(http.HandlerFunc(serveInstructors(instructorRepo)))))
	mux.Handle("GET /admin/events", csrfMiddleware(adminMiddleware(http.HandlerFunc(serveEvents(userRepo, eventRepo)))))
	mux.Handle("GET /admin/survey/questions", csrfMiddleware(adminMiddleware(http.HandlerFunc(serveSurveyQuestions))))
	mux.Handle("GET /admin/survey/results", csrfMiddleware(adminMiddleware(http.HandlerFunc(serveSurveyResults))))
	mux.Handle("GET /admin/user-view", csrfMiddleware(adminMiddleware(http.HandlerFunc(serveUserView))))

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
	mux.Handle("/", csrfMiddleware(authMiddleware(http.HandlerFunc(serveRoot()))))

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
	if user == nil || user.Role != models.RoleAdmin {
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
	if user == nil || user.Role != models.RoleAdmin {
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
		if user == nil || user.Role != models.RoleAdmin {
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

func serveUserView(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil || user.Role != models.RoleAdmin {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}

	// Create mock user data for simulation
	mockUser := &models.User{
		ID:                "mock-user-id",
		FirstName:         "Mario",
		LastName:          "Rossi",
		Email:             "mario.rossi@example.com",
		SubType:           models.SubTypeShared,
		RemainingAccesses: 8,
		ExpiresAt:         time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC),
		Goals:             sql.NullString{String: "Migliorare forma fisica e benessere generale", Valid: true},
	}

	// Create mock bookings
	type BookingDisplay struct {
		ID                int64
		StartsAt          string
		StartsAtFormatted string
		CreatedAt         string
		InstructorName    string
	}

	mockBookings := []BookingDisplay{
		{
			ID:                1,
			StartsAt:          time.Date(2025, 11, 2, 9, 0, 0, 0, time.UTC).Format(time.RFC3339),
			StartsAtFormatted: "02 Nov 2025, 09:00",
			CreatedAt:         time.Date(2025, 10, 20, 10, 30, 0, 0, time.UTC).Format(time.RFC3339),
			InstructorName:    "Luca Bianchi",
		},
		{
			ID:                2,
			StartsAt:          time.Date(2025, 11, 5, 14, 30, 0, 0, time.UTC).Format(time.RFC3339),
			StartsAtFormatted: "05 Nov 2025, 14:30",
			CreatedAt:         time.Date(2025, 10, 21, 15, 0, 0, 0, time.UTC).Format(time.RFC3339),
			InstructorName:    "Anna Verdi",
		},
		{
			ID:                3,
			StartsAt:          time.Date(2025, 11, 10, 11, 0, 0, 0, time.UTC).Format(time.RFC3339),
			StartsAtFormatted: "10 Nov 2025, 11:00",
			CreatedAt:         time.Date(2025, 10, 22, 9, 15, 0, 0, time.UTC).Format(time.RFC3339),
			InstructorName:    "Luca Bianchi",
		},
	}

	data := map[string]interface{}{
		"User":              mockUser,
		"ExpiresAt":         mockUser.ExpiresAt.Format("02 Jan 2006"),
		"RemainingAccesses": mockUser.RemainingAccesses,
		"Bookings":          mockBookings,
		"IsSimulation":      true,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tpl.ExecuteTemplate(w, "index.html", data); err != nil {
		log.Print(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}
