package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/alarmfox/wellness-nutrition/app/mail"
	"github.com/alarmfox/wellness-nutrition/app/middleware"
	"github.com/alarmfox/wellness-nutrition/app/models"
)

type BookingHandler struct {
	bookingRepo *models.BookingRepository
	slotRepo    *models.SlotRepository
	eventRepo   *models.EventRepository
	userRepo    *models.UserRepository
	mailer      *mail.Mailer
}

func NewBookingHandler(
	bookingRepo *models.BookingRepository,
	slotRepo *models.SlotRepository,
	eventRepo *models.EventRepository,
	userRepo *models.UserRepository,
	mailer *mail.Mailer,
) *BookingHandler {
	return &BookingHandler{
		bookingRepo: bookingRepo,
		slotRepo:    slotRepo,
		eventRepo:   eventRepo,
		userRepo:    userRepo,
		mailer:      mailer,
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
	StartsAt string `json:"startsAt"`
}

func (h *BookingHandler) Create(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		sendJSON(w, http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
		return
	}
	
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
	
	startsAt, err := time.Parse(time.RFC3339, req.StartsAt)
	if err != nil {
		sendJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid date format"})
		return
	}
	
	// Check if slot exists and is not disabled
	slot, err := h.slotRepo.GetByTime(startsAt)
	if err != nil {
		if err == sql.ErrNoRows {
			sendJSON(w, http.StatusBadRequest, map[string]string{"error": "Slot not found"})
			return
		}
		log.Printf("Error getting slot: %v", err)
		sendJSON(w, http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
		return
	}
	
	if slot.Disabled {
		sendJSON(w, http.StatusBadRequest, map[string]string{"error": "Slot is disabled"})
		return
	}
	
	// Create booking
	booking := &models.Booking{
		UserID:    user.ID,
		CreatedAt: time.Now(),
		StartsAt:  startsAt,
	}
	
	if err := h.bookingRepo.Create(booking); err != nil {
		log.Printf("Error creating booking: %v", err)
		sendJSON(w, http.StatusConflict, map[string]string{"error": "Booking already exists"})
		return
	}
	
	// Update slot people count
	if err := h.slotRepo.IncrementPeopleCount(startsAt); err != nil {
		log.Printf("Error updating slot: %v", err)
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
	
	sendJSON(w, http.StatusCreated, booking)
}

type DeleteBookingRequest struct {
	ID       string `json:"id"`
	StartsAt string `json:"startsAt"`
}

func (h *BookingHandler) Delete(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		sendJSON(w, http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
		return
	}
	
	var req DeleteBookingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request"})
		return
	}
	
	id, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		sendJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid booking ID"})
		return
	}
	
	// Get booking to verify ownership
	booking, err := h.bookingRepo.GetByID(id)
	if err != nil {
		if err == sql.ErrNoRows {
			sendJSON(w, http.StatusNotFound, map[string]string{"error": "Booking not found"})
			return
		}
		log.Printf("Error getting booking: %v", err)
		sendJSON(w, http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
		return
	}
	
	if booking.UserID != user.ID {
		sendJSON(w, http.StatusForbidden, map[string]string{"error": "Forbidden"})
		return
	}
	
	// Delete booking
	if err := h.bookingRepo.Delete(id); err != nil {
		log.Printf("Error deleting booking: %v", err)
		sendJSON(w, http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
		return
	}
	
	// Update slot people count
	if err := h.slotRepo.DecrementPeopleCount(booking.StartsAt); err != nil {
		log.Printf("Error updating slot: %v", err)
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
	
	sendJSON(w, http.StatusOK, map[string]string{"message": "Booking deleted successfully"})
}

func (h *BookingHandler) GetAvailableSlots(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		sendJSON(w, http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
		return
	}
	
	// Get slots for the next 3 months
	from := time.Now()
	to := from.AddDate(0, 3, 0)
	
	slots, err := h.slotRepo.GetAvailableSlots(from, to)
	if err != nil {
		log.Printf("Error getting slots: %v", err)
		sendJSON(w, http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
		return
	}
	
	// Convert to Unix timestamps
	timestamps := make([]int64, len(slots))
	for i, slot := range slots {
		timestamps[i] = slot.StartsAt.Unix()
	}
	
	sendJSON(w, http.StatusOK, timestamps)
}

// GetAllBookings returns all bookings for admin calendar view
func (h *BookingHandler) GetAllBookings(w http.ResponseWriter, r *http.Request) {
	// Get date range from query parameters
	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")
	
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
	
	bookings, err := h.bookingRepo.GetByDateRange(from, to)
	if err != nil {
		log.Printf("Error getting bookings: %v", err)
		sendJSON(w, http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
		return
	}
	
	// Enrich bookings with user information
	type BookingWithUser struct {
		ID        int64     `json:"id"`
		StartsAt  time.Time `json:"startsAt"`
		CreatedAt time.Time `json:"createdAt"`
		User      struct {
			ID        string `json:"id"`
			FirstName string `json:"firstName"`
			LastName  string `json:"lastName"`
			SubType   string `json:"subType"`
		} `json:"user"`
	}
	
	result := make([]BookingWithUser, len(bookings))
	for i, booking := range bookings {
		user, err := h.userRepo.GetByID(booking.UserID)
		if err != nil {
			log.Printf("Error getting user for booking: %v", err)
			continue
		}
		
		result[i] = BookingWithUser{
			ID:        booking.ID,
			StartsAt:  booking.StartsAt,
			CreatedAt: booking.CreatedAt,
		}
		result[i].User.ID = user.ID
		result[i].User.FirstName = user.FirstName
		result[i].User.LastName = user.LastName
		result[i].User.SubType = string(user.SubType)
	}
	
	sendJSON(w, http.StatusOK, result)
}
