package handlers

import (
	"log"
	"net/http"

	"github.com/alarmfox/wellness-nutrition/app/middleware"
	"github.com/alarmfox/wellness-nutrition/app/models"
)

type PageHandler struct {
	userRepo    *models.UserRepository
	bookingRepo *models.BookingRepository
	eventRepo   *models.EventRepository
}

func NewPageHandler(
	userRepo *models.UserRepository,
	bookingRepo *models.BookingRepository,
	eventRepo *models.EventRepository,
) *PageHandler {
	return &PageHandler{
		userRepo:    userRepo,
		bookingRepo: bookingRepo,
		eventRepo:   eventRepo,
	}
}

func (h *PageHandler) ServeUsers(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil || user.Role != models.RoleAdmin {
		http.Redirect(w, r, "/signin", http.StatusSeeOther)
		return
	}
	
	users, err := h.userRepo.GetAll()
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
		SubType             string
		EmailVerified       bool
		ExpiresAtFormatted  string
		RemainingAccesses   int
	}
	
	var displayUsers []UserDisplay
	for _, u := range users {
		displayUsers = append(displayUsers, UserDisplay{
			ID:                  u.ID,
			FirstName:           u.FirstName,
			LastName:            u.LastName,
			Email:               u.Email,
			SubType:             string(u.SubType),
			EmailVerified:       u.EmailVerified.Valid,
			ExpiresAtFormatted:  u.ExpiresAt.Format("02 Jan 2006"),
			RemainingAccesses:   u.RemainingAccesses,
		})
	}
	
	_ = displayUsers // Unused for now - will be moved to main.go
	
	// Render template - this will be done in main.go
	w.Header().Set("X-Template", "users.html")
	w.Header().Set("X-Data", "users")
	
	// For now, we'll pass this through context or use a different approach
	// This will be handled by updating main.go to support template rendering
}

func (h *PageHandler) ServeEvents(w http.ResponseWriter, r *http.Request) {
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
		ID                  int
		UserName            string
		Type                string
		OccurredAtFormatted string
		StartsAtFormatted   string
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
			ID:                  e.ID,
			UserName:            userName,
			Type:                string(e.Type),
			OccurredAtFormatted: e.OccurredAt.Format("02 Jan 2006 15:04"),
			StartsAtFormatted:   e.StartsAt.Format("02 Jan 2006 15:04"),
		})
	}
	
	_ = displayEvents // Unused for now - will be moved to main.go
	
	// Render template - this will be done in main.go
	w.Header().Set("X-Template", "events.html")
	w.Header().Set("X-Data", "events")
}
