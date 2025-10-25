package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/alarmfox/wellness-nutrition/app/mail"
	"github.com/alarmfox/wellness-nutrition/app/middleware"
	"github.com/alarmfox/wellness-nutrition/app/models"
	"github.com/alarmfox/wellness-nutrition/app/websocket"
)

type BookingHandler struct {
	bookingRepo    *models.BookingRepository
	eventRepo      *models.EventRepository
	userRepo       *models.UserRepository
	instructorRepo *models.InstructorRepository
	mailer         *mail.Mailer
	hub            *websocket.Hub
}

func NewBookingHandler(
	bookingRepo *models.BookingRepository,
	eventRepo *models.EventRepository,
	userRepo *models.UserRepository,
	instructorRepo *models.InstructorRepository,
	mailer *mail.Mailer,
	hub *websocket.Hub,
) *BookingHandler {
	return &BookingHandler{
		bookingRepo:    bookingRepo,
		eventRepo:      eventRepo,
		userRepo:       userRepo,
		instructorRepo: instructorRepo,
		mailer:         mailer,
		hub:            hub,
	}
}

func (h *BookingHandler) GetCurrent(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		sendJSON(w, http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
		return
	}

	bookings, err := h.bookingRepo.GetByUserID(user.ID)
	if err != nil {
		log.Printf("Error getting bookings: %v", err)
		sendJSON(w, http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
		return
	}

	sendJSON(w, http.StatusOK, bookings)
}

type CreateBookingRequest struct {
	StartsAt     string `json:"startsAt"`
	InstructorID int64  `json:"instructorId"`
}

func (h *BookingHandler) Create(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())

	// Check if user can create booking
	if time.Now().After(user.ExpiresAt) || user.RemainingAccesses <= 0 {
		sendJSON(w, http.StatusUnauthorized, map[string]string{"error": "Subscription expired or no remaining accesses"})
		return
	}

	var req CreateBookingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request"})
		return
	}

	startsAt, _ := time.Parse(time.RFC3339, req.StartsAt)

	booking := models.Booking{
		InstructorID: req.InstructorID,
		UserID:       sql.NullString{Valid: true, String: user.ID},
		StartsAt:     startsAt,
		Type:         models.BookingTypeSimple,
	}

	if err := h.bookingRepo.Create(&booking); err != nil {
		log.Printf("Error creating booking: %v", err)
		sendJSON(w, http.StatusConflict, map[string]string{"error": "Booking already exists"})
		return
	}

	if err := h.userRepo.DecrementAccesses(user.ID); err != nil {
		log.Printf("Error decrementing accesses: %v", err)
		sendJSON(w, http.StatusConflict, map[string]string{"error": "Cannot decrement accesses"})

	}

	// Create event
	event := &models.Event{
		UserID:     user.ID,
		StartsAt:   startsAt,
		Type:       models.EventTypeCreated,
		OccurredAt: time.Now(),
	}
	if err := h.eventRepo.Create(event); err != nil {
		log.Printf("Error creating event: %v", err)
	}

	// Send notification email
	go func() {
		if err := h.mailer.SendNewBookingNotification(user.FirstName, user.LastName, startsAt); err != nil {
			log.Printf("Error sending notification: %v", err)
		}
	}()
	//
	// // Send WebSocket notification
	if h.hub != nil {
		h.hub.BroadcastJSON(
			websocket.NotificationBookingCreated,
			fmt.Sprintf("Nuova prenotazione: %s %s - %s", user.FirstName, user.LastName, startsAt.Format("02/01/2006 15:04")),
			fmt.Sprintf("%s %s", user.FirstName, user.LastName),
			startsAt.Format("02/01/2006 15:04"),
		)
	}

	sendJSON(w, http.StatusCreated, booking)
}

func (h *BookingHandler) Delete(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	id := r.PathValue("id")

	idInt, err := strconv.ParseInt(id, 10, 32)
	if err != nil {
		sendJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid ID"})
		return
	}

	// Get booking to verify ownership
	booking, err := h.bookingRepo.GetByID(idInt)
	if err != nil {
		if err == sql.ErrNoRows {
			sendJSON(w, http.StatusNotFound, map[string]string{"error": "Booking not found"})
			return
		}
		log.Printf("Error getting booking: %v", err)
		sendJSON(w, http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
		return
	}

	// Check if user is admin or owns the booking
	if user == nil || (user.Role != models.RoleAdmin && booking.UserID.String != user.ID) {
		sendJSON(w, http.StatusForbidden, map[string]string{"error": "Forbidden"})
		return
	}

	// Delete booking
	if err := h.bookingRepo.Delete(idInt); err != nil {
		log.Printf("Error deleting booking: %v", err)
		sendJSON(w, http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
		return
	}

	// Refund policy: only refund if deleted 3+ hours before event
	shouldRefund := time.Until(booking.StartsAt) >= 3*time.Hour

	// Only increment accesses if cancelling 3+ hours before
	if shouldRefund {
		if err := h.userRepo.IncrementAccesses(booking.UserID.String); err != nil {
			log.Printf("Error incrementing accesses: %v", err)
		}
	}

	// Create event
	event := &models.Event{
		UserID:     user.ID,
		StartsAt:   booking.StartsAt,
		Type:       models.EventTypeDeleted,
		OccurredAt: time.Now(),
	}
	if err := h.eventRepo.Create(event); err != nil {
		log.Printf("Error creating event: %v", err)
	}

	// Send notification email
	go func() {
		if err := h.mailer.SendDeleteBookingNotification(user.FirstName, user.LastName, booking.StartsAt); err != nil {
			log.Printf("Error sending notification: %v", err)
		}
	}()

	// Send WebSocket notification
	if h.hub != nil {
		// Get user info for the notification
		bookingUser, _ := h.userRepo.GetByID(booking.UserID.String)
		userName := "Unknown"
		if bookingUser != nil {
			userName = fmt.Sprintf("%s %s", bookingUser.FirstName, bookingUser.LastName)
		}
		h.hub.BroadcastJSON(
			websocket.NotificationBookingDeleted,
			fmt.Sprintf("Prenotazione cancellata: %s - %s", userName, booking.StartsAt.Format("02/01/2006 15:04")),
			userName,
			booking.StartsAt.Format("02/01/2006 15:04"),
		)
	}

	sendJSON(w, http.StatusOK, map[string]string{"message": "Booking deleted successfully"})
}

func (h *BookingHandler) DeleteAdmin(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	idInt, err := strconv.ParseInt(id, 10, 32)
	if err != nil {
		sendJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid ID"})
		return
	}

	refund := r.URL.Query().Get("refund")
	if refund == "" {
		refund = "false"
	}

	refundBool, err := strconv.ParseBool(r.URL.Query().Get("refund"))
	if err != nil {
		log.Printf("Error deleting booking: %v", err)
		sendJSON(w, http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
	}

	booking, err := h.bookingRepo.GetByID(idInt)
	if err != nil {
		log.Printf("Error deleting booking: %v", err)
		sendJSON(w, http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
		return
	}

	// Delete booking
	if err := h.bookingRepo.Delete(idInt); err != nil {
		log.Printf("Error deleting booking: %v", err)
		sendJSON(w, http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
		return
	}

	if booking.Type == models.BookingTypeSimple && refundBool {
		h.userRepo.IncrementAccesses(booking.UserID.String)
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetAllBookings returns all bookings for admin calendar view
func (h *BookingHandler) GetAllBookings(w http.ResponseWriter, r *http.Request) {
	// Get date range from query parameters
	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")
	instructorID := r.URL.Query().Get("instructorId")

	var from, to time.Time
	var err error

	if fromStr != "" {
		from, err = time.Parse(time.RFC3339, fromStr)
		if err != nil {
			from = time.Now().AddDate(0, 0, -7) // Default: 1 week ago
		}
	} else {
		from = time.Now().AddDate(0, 0, -7)
	}

	if toStr != "" {
		to, err = time.Parse(time.RFC3339, toStr)
		if err != nil {
			to = time.Now().AddDate(0, 1, 0) // Default: 1 month ahead
		}
	} else {
		to = time.Now().AddDate(0, 1, 0)
	}

	var bookings []*models.Booking

	// Filter by instructor if specified
	if instructorID != "" {
		bookings, err = h.bookingRepo.GetByInstructorAndDateRange(instructorID, from, to)
	} else {
		bookings, err = h.bookingRepo.GetByDateRange(from, to)
	}

	if err != nil {
		log.Printf("Error getting bookings: %v", err)
		sendJSON(w, http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
		return
	}

	// Enrich bookings with user and instructor information
	type BookingWithUser struct {
		ID           int64     `json:"id"`
		StartsAt     time.Time `json:"startsAt"`
		CreatedAt    time.Time `json:"createdAt"`
		InstructorId int64     `json:"instructorId"`
		Type         string    `json:"type"`
		User         *struct {
			ID        string `json:"id"`
			FirstName string `json:"firstName"`
			LastName  string `json:"lastName"`
			Email     string `json:"email"`
			SubType   string `json:"subType"`
		} `json:"user,omitempty"`
	}

	result := make([]BookingWithUser, len(bookings))
	for i, booking := range bookings {
		if booking.UserID.Valid {
			user, err := h.userRepo.GetByID(booking.UserID.String)
			if err != nil {
				log.Printf("Error getting user %s: %v", booking.UserID.String, err)
				continue
			}

			result[i].User = &struct {
				ID        string `json:"id"`
				FirstName string `json:"firstName"`
				LastName  string `json:"lastName"`
				Email     string `json:"email"`
				SubType   string `json:"subType"`
			}{
				ID:        user.ID,
				FirstName: user.FirstName,
				LastName:  user.LastName,
				Email:     user.Email,
				SubType:   string(user.SubType),
			}

		}
		result[i].ID = booking.ID
		result[i].StartsAt = booking.StartsAt
		result[i].CreatedAt = booking.CreatedAt

		result[i].InstructorId = booking.InstructorID
		result[i].Type = string(booking.Type)
	}

	sendJSON(w, http.StatusOK, result)
}

// CreateBookingForUser allows admin to create a booking for a specific user
func (h *BookingHandler) CreateBookingForUser(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID       string             `json:"userId"`
		StartsAt     string             `json:"startsAt"`
		InstructorID int64              `json:"instructorId"`
		Type         models.BookingType `json:"type"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request"})
		return
	}

	startsAt, err := time.Parse(time.RFC3339, req.StartsAt)
	if err != nil {
		sendJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid date format"})
		return
	}

	// Create booking
	booking := &models.Booking{
		UserID:       sql.NullString{Valid: req.UserID != "", String: req.UserID},
		InstructorID: req.InstructorID,
		StartsAt:     startsAt,
		Type:         req.Type,
	}

	if err := h.bookingRepo.Create(booking); err != nil {
		log.Printf("Error creating booking: %v", err)
		sendJSON(w, http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
		return
	}

	if booking.Type == models.BookingTypeSimple {
		user, err := h.userRepo.GetByID(req.UserID)
		if err != nil {
			log.Printf("Error getting user: %v", err)
			sendJSON(w, http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
			return
		}

		err = h.userRepo.DecrementAccesses(req.UserID)
		if err != nil {
			log.Printf("Error decrementing access: %v", err)
			sendJSON(w, http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
			return
		}

		// Send notification email (synchronous)
		if err := h.mailer.SendNewBookingNotification(user.FirstName, user.LastName, startsAt); err != nil {
			log.Printf("Error sending notification: %v", err)
		}
	}

	sendJSON(w, http.StatusCreated, map[string]interface{}{
		"message":   "Booking created successfully",
		"bookingId": booking.ID,
	})
}
