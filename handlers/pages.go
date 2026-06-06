package handlers

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/alarmfox/wellness-nutrition/app/middleware"
	"github.com/alarmfox/wellness-nutrition/app/models"
)

type PageHandler struct {
	userRepo       *models.UserRepository
	bookingRepo    *models.BookingRepository
	eventRepo      *models.EventRepository
	instructorRepo *models.InstructorRepository
	questionRepo   *models.QuestionRepository
	tpl            *template.Template
}

func NewPageHandler(
	userRepo *models.UserRepository,
	bookingRepo *models.BookingRepository,
	eventRepo *models.EventRepository,
	instructorRepo *models.InstructorRepository,
	questionRepo *models.QuestionRepository,
	tpl *template.Template,
) *PageHandler {
	return &PageHandler{
		userRepo:       userRepo,
		bookingRepo:    bookingRepo,
		eventRepo:      eventRepo,
		instructorRepo: instructorRepo,
		questionRepo:   questionRepo,
		tpl:            tpl,
	}
}

func (h *PageHandler) ServeRoot(w http.ResponseWriter, r *http.Request) {
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

func (h *PageHandler) ServeUserDashboard(w http.ResponseWriter, r *http.Request) {
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
	bookings, err := h.bookingRepo.GetByUserID(user.ID)
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
	loc, err := time.LoadLocation(businessTimeZone)
	if err != nil {
		panic(err)
	}
	for _, b := range bookings {
		instructorName := ""
		instructor, err := h.instructorRepo.GetByID(b.InstructorID)
		if err == nil {
			instructorName = instructor.FirstName
			if instructor.LastName != "" {
				instructorName += " " + instructor.LastName
			}
		}

		startsAt := b.StartsAt.In(loc)
		displayBookings = append(displayBookings, BookingDisplay{
			ID:                b.ID,
			StartsAt:          b.StartsAt.Format(time.RFC3339),
			StartsAtFormatted: startsAt.Format("02 Jan 2006, 15:04"),
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
	if err := h.tpl.ExecuteTemplate(w, "index.html", data); err != nil {
		log.Print(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func (h *PageHandler) ServeAdminHome(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil || user.Role != models.RoleAdmin {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}

	// Redirect admin to calendar
	http.Redirect(w, r, "/admin/calendar", http.StatusSeeOther)
}

func (h *PageHandler) ServeCalendar(w http.ResponseWriter, r *http.Request) {
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
	if err := h.tpl.ExecuteTemplate(w, "calendar.html", nil); err != nil {
		log.Print(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func (h *PageHandler) ServeSignIn(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Error": r.URL.Query().Get("error"),
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tpl.ExecuteTemplate(w, "signin.html", data); err != nil {
		log.Print(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func (h *PageHandler) ServeReset(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Token":   r.URL.Query().Get("token"),
		"Success": r.URL.Query().Get("success"),
		"Error":   r.URL.Query().Get("error"),
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tpl.ExecuteTemplate(w, "reset.html", data); err != nil {
		log.Print(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func (h *PageHandler) ServeVerify(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	data := map[string]interface{}{
		"Token": token,
		"Error": r.URL.Query().Get("error"),
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tpl.ExecuteTemplate(w, "verify.html", data); err != nil {
		log.Print(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func (h *PageHandler) ServeUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.userRepo.GetAll()
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
	if err := h.tpl.ExecuteTemplate(w, "users.html", data); err != nil {
		log.Print(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func (h *PageHandler) ServeEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	user := middleware.GetUserFromContext(r.Context())
	if user == nil || user.Role != models.RoleAdmin {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}

	events, err := h.eventRepo.GetAll()
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
		u, err := h.userRepo.GetByID(e.UserID)
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
	if err := h.tpl.ExecuteTemplate(w, "events.html", data); err != nil {
		log.Print(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func (h *PageHandler) ServeSurvey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	questions, err := h.questionRepo.GetAll()
	if err != nil {
		log.Printf("Error getting questions: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Questions": questions,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tpl.ExecuteTemplate(w, "survey.html", data); err != nil {
		log.Print(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func (h *PageHandler) ServeSurveyThanks(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tpl.ExecuteTemplate(w, "survey-thanks.html", nil); err != nil {
		log.Print(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func (h *PageHandler) ServeSurveyQuestions(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil || user.Role != models.RoleAdmin {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tpl.ExecuteTemplate(w, "survey-questions.html", nil); err != nil {
		log.Print(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func (h *PageHandler) ServeSurveyResults(w http.ResponseWriter, r *http.Request) {
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
	if err := h.tpl.ExecuteTemplate(w, "survey-results.html", nil); err != nil {
		log.Print(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func (h *PageHandler) ServeInstructors(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	user := middleware.GetUserFromContext(r.Context())
	if user == nil || user.Role != models.RoleAdmin {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}

	instructors, err := h.instructorRepo.GetAll()
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
		MaxSlots  int
		Enabled   bool
		CreatedAt string
	}

	var displayInstructors []InstructorDisplay
	for _, i := range instructors {
		displayInstructors = append(displayInstructors, InstructorDisplay{
			ID:        i.ID,
			FirstName: i.FirstName,
			LastName:  i.LastName,
			MaxSlots:  i.MaxSlots,
			Enabled:   i.Enabled,
			CreatedAt: i.CreatedAt.Format("02 Jan 2006"),
		})
	}

	data := map[string]interface{}{
		"Instructors": displayInstructors,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tpl.ExecuteTemplate(w, "instructors.html", data); err != nil {
		log.Print(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func (h *PageHandler) ServeUserView(w http.ResponseWriter, r *http.Request) {
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
	if err := h.tpl.ExecuteTemplate(w, "index.html", data); err != nil {
		log.Print(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}
