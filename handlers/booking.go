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
	bookingRepo         *models.BookingRepository
	slotRepo            *models.SlotRepository
	eventRepo           *models.EventRepository
	userRepo            *models.UserRepository
	instructorRepo      *models.InstructorRepository
	instructorSlotRepo  *models.InstructorSlotRepository
	mailer              *mail.Mailer
	hub                 *websocket.Hub
}

func NewBookingHandler(
	bookingRepo *models.BookingRepository,
	slotRepo *models.SlotRepository,
	eventRepo *models.EventRepository,
	userRepo *models.UserRepository,
	instructorRepo *models.InstructorRepository,
	instructorSlotRepo *models.InstructorSlotRepository,
	mailer *mail.Mailer,
	hub *websocket.Hub,
) *BookingHandler {
	return &BookingHandler{
		bookingRepo:        bookingRepo,
		slotRepo:           slotRepo,
		eventRepo:          eventRepo,
		userRepo:           userRepo,
		instructorRepo:     instructorRepo,
		instructorSlotRepo: instructorSlotRepo,
		mailer:             mailer,
		hub:                hub,
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
	InstructorID string `json:"instructorId"`
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

	// Validate instructor selection
	if req.InstructorID == "" {
		sendJSON(w, http.StatusBadRequest, map[string]string{"error": "Instructor selection is required"})
		return
	}

	// Verify instructor exists
	_, err := h.instructorRepo.GetByID(req.InstructorID)
	if err != nil {
		if err == sql.ErrNoRows {
			sendJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid instructor"})
			return
		}
		log.Printf("Error getting instructor: %v", err)
		sendJSON(w, http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
		return
	}

	startsAt, err := time.Parse(time.RFC3339, req.StartsAt)
	if err != nil {
		sendJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid date format"})
		return
	}

	// Check if slot exists and is not disabled
	// If slot doesn't exist, create it (lazy creation)
	slot, err := h.slotRepo.GetByTime(startsAt)
	if err != nil {
		if err == sql.ErrNoRows {
			// Lazy create slot
			slot = &models.Slot{
				StartsAt:    startsAt,
				PeopleCount: 0,
				Disabled:    false,
			}
			if err := h.slotRepo.Create(slot); err != nil {
				log.Printf("Error creating slot: %v", err)
				sendJSON(w, http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
				return
			}
		} else {
			log.Printf("Error getting slot: %v", err)
			sendJSON(w, http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
			return
		}
	}

	if slot.Disabled {
		sendJSON(w, http.StatusBadRequest, map[string]string{"error": "Slot is disabled"})
		return
	}

	// Check instructor slot capacity (max 2 people per instructor per slot)
	instructorSlot, err := h.instructorSlotRepo.GetByInstructorAndTime(req.InstructorID, startsAt)
	if err != nil {
		if err == sql.ErrNoRows {
			// Create instructor slot
			instructorSlot = &models.InstructorSlot{
				InstructorID: req.InstructorID,
				StartsAt:     startsAt,
				PeopleCount:  0,
				MaxCapacity:  2,
			}
			if err := h.instructorSlotRepo.Create(instructorSlot); err != nil {
				log.Printf("Error creating instructor slot: %v", err)
				sendJSON(w, http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
				return
			}
		} else {
			log.Printf("Error getting instructor slot: %v", err)
			sendJSON(w, http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
			return
		}
	}

	// Check if instructor slot is full
	if instructorSlot.PeopleCount >= instructorSlot.MaxCapacity {
		sendJSON(w, http.StatusBadRequest, map[string]string{"error": "This instructor's slot is full"})
		return
	}

	// Create booking
	booking := &models.Booking{
		UserID:       user.ID,
		InstructorID: sql.NullString{String: req.InstructorID, Valid: true},
		CreatedAt:    time.Now(),
		StartsAt:     startsAt,
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

	// Update instructor slot people count
	if err := h.instructorSlotRepo.IncrementPeopleCount(req.InstructorID, startsAt); err != nil {
		log.Printf("Error updating instructor slot: %v", err)
	}

	// Decrement user remaining accesses
	if err := h.userRepo.DecrementAccesses(user.ID); err != nil {
		log.Printf("Error decrementing accesses: %v", err)
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
		if err := h.mailer.SendNewBookingNotification(user.FirstName, user.LastName.String, startsAt); err != nil {
			log.Printf("Error sending notification: %v", err)
		}
	}()

	// Send WebSocket notification
	if h.hub != nil {
		h.hub.BroadcastJSON(
			websocket.NotificationBookingCreated,
			fmt.Sprintf("Nuova prenotazione: %s %s - %s", user.FirstName, user.LastName.String, startsAt.Format("02/01/2006 15:04")),
			fmt.Sprintf("%s %s", user.FirstName, user.LastName.String),
			startsAt.Format("02/01/2006 15:04"),
		)
	}

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

	// Check if user is admin or owns the booking
	admin := middleware.GetAdminFromContext(r.Context())
	if admin == nil && (user == nil || booking.UserID != user.ID) {
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

	// Update instructor slot people count if instructor was assigned
	if booking.InstructorID.Valid {
		if err := h.instructorSlotRepo.DecrementPeopleCount(booking.InstructorID.String, booking.StartsAt); err != nil {
			log.Printf("Error updating instructor slot: %v", err)
		}
	}

	// Refund policy: only refund if deleted 3+ hours before event
	timeUntilBooking := booking.StartsAt.Sub(time.Now())
	shouldRefund := timeUntilBooking >= 3*time.Hour

	// Only increment accesses if cancelling 3+ hours before
	if shouldRefund {
		if err := h.userRepo.IncrementAccesses(booking.UserID); err != nil {
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
		if err := h.mailer.SendDeleteBookingNotification(user.FirstName, user.LastName.String, booking.StartsAt); err != nil {
			log.Printf("Error sending notification: %v", err)
		}
	}()

	// Send WebSocket notification
	if h.hub != nil {
		// Get user info for the notification
		bookingUser, _ := h.userRepo.GetByID(booking.UserID)
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

func (h *BookingHandler) GetAvailableSlots(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		sendJSON(w, http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
		return
	}

	// Get instructor ID from query parameters (optional)
	instructorID := r.URL.Query().Get("instructorId")

	// Get slots for the next 30 days
	from := time.Now()
	to := from.AddDate(0, 1, 0)
	if user.ExpiresAt.Before(to) {
		to = user.ExpiresAt
	}

	// If instructor is specified, get instructor-specific available slots
	if instructorID != "" {
		instructorSlots, err := h.instructorSlotRepo.GetAvailableForInstructor(instructorID, from, to)
		if err != nil {
			log.Printf("Error getting instructor slots: %v", err)
			sendJSON(w, http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
			return
		}

		// Convert to response format
		type SlotResponse struct {
			StartsAt      string `json:"StartsAt"`
			PeopleCount   int    `json:"PeopleCount"`
			MaxCapacity   int    `json:"MaxCapacity"`
			InstructorID  string `json:"InstructorId"`
		}

		response := make([]SlotResponse, 0, len(instructorSlots))
		for _, slot := range instructorSlots {
			response = append(response, SlotResponse{
				StartsAt:     slot.StartsAt.Format(time.RFC3339),
				PeopleCount:  slot.PeopleCount,
				MaxCapacity:  slot.MaxCapacity,
				InstructorID: slot.InstructorID,
			})
		}

		sendJSON(w, http.StatusOK, map[string]interface{}{
			"slots": response,
		})
		return
	}

	// Otherwise, get all available slots
	slots, err := h.slotRepo.GetAvailableSlots(from, to)
	if err != nil {
		log.Printf("Error getting slots: %v", err)
		sendJSON(w, http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
		return
	}

	// Convert to response format with slot details
	type SlotResponse struct {
		StartsAt    string `json:"StartsAt"`
		PeopleCount int    `json:"PeopleCount"`
		Disabled    bool   `json:"Disabled"`
	}

	response := make([]SlotResponse, 0, len(slots))
	for _, slot := range slots {
		response = append(response, SlotResponse{
			StartsAt:    slot.StartsAt.Format(time.RFC3339),
			PeopleCount: slot.PeopleCount,
			Disabled:    slot.Disabled,
		})
	}

	sendJSON(w, http.StatusOK, map[string]interface{}{
		"slots": response,
	})
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
		ID        int64     `json:"ID"`
		StartsAt  time.Time `json:"StartsAt"`
		CreatedAt time.Time `json:"CreatedAt"`
		User      struct {
			ID        string `json:"ID"`
			FirstName string `json:"FirstName"`
			LastName  string `json:"LastName"`
			Email     string `json:"Email"`
			SubType   string `json:"SubType"`
		} `json:"User"`
		Instructor *struct {
			ID        string `json:"ID"`
			FirstName string `json:"FirstName"`
			LastName  string `json:"LastName"`
		} `json:"Instructor,omitempty"`
	}

	result := make([]BookingWithUser, len(bookings))
	for i, booking := range bookings {
		user, err := h.userRepo.GetByID(booking.UserID)
		if err != nil {
			log.Printf("Error getting user %s: %v", booking.UserID, err)
			continue
		}

		result[i].ID = booking.ID
		result[i].StartsAt = booking.StartsAt
		result[i].CreatedAt = booking.CreatedAt
		result[i].User.ID = user.ID
		result[i].User.FirstName = user.FirstName
		result[i].User.LastName = user.LastName.String
		result[i].User.Email = user.Email
		result[i].User.SubType = string(user.SubType)

		// Add instructor info if available
		if booking.InstructorID.Valid {
			instructor, err := h.instructorRepo.GetByID(booking.InstructorID.String)
			if err == nil {
				result[i].Instructor = &struct {
					ID        string `json:"ID"`
					FirstName string `json:"FirstName"`
					LastName  string `json:"LastName"`
				}{
					ID:        instructor.ID,
					FirstName: instructor.FirstName,
					LastName:  instructor.LastName.String,
				}
			}
		}
	}

	sendJSON(w, http.StatusOK, result)
}

// GetAllSlots returns all slots (including disabled) for admin calendar view
func (h *BookingHandler) GetAllSlots(w http.ResponseWriter, r *http.Request) {
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

	slots, err := h.slotRepo.GetSlotsByDateRange(from, to)
	if err != nil {
		log.Printf("Error getting slots: %v", err)
		sendJSON(w, http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
		return
	}

	// Convert to JSON-friendly format
	type SlotResponse struct {
		StartsAt    time.Time `json:"StartsAt"`
		PeopleCount int       `json:"PeopleCount"`
		Disabled    bool      `json:"Disabled"`
	}

	result := make([]SlotResponse, len(slots))
	for i, slot := range slots {
		result[i].StartsAt = slot.StartsAt
		result[i].PeopleCount = slot.PeopleCount
		result[i].Disabled = slot.Disabled
	}

	sendJSON(w, http.StatusOK, result)
}

// DisableSlot marks a slot as unavailable (with booking check)
func (h *BookingHandler) DisableSlot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method not allowed"})
		return
	}

	var req struct {
		StartsAt string `json:"startsAt"`
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

	// Check if slot exists
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

	// Check if slot has bookings
	bookings, err := h.bookingRepo.GetBySlotTime(startsAt)
	if err != nil {
		log.Printf("Error getting bookings for slot: %v", err)
		// Continue anyway, don't fail on this
		bookings = []*models.Booking{}
	}

	if len(bookings) > 0 {
		// Return info about bookings that need confirmation
		sendJSON(w, http.StatusOK, map[string]interface{}{
			"hasBookings":  true,
			"bookingCount": len(bookings),
			"message":      "This slot has bookings",
		})
		return
	}

	// No bookings, proceed to disable
	slot.Disabled = true
	if err := h.slotRepo.Update(slot); err != nil {
		log.Printf("Error updating slot: %v", err)
		sendJSON(w, http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
		return
	}

	// Create event log
	user := middleware.GetUserFromContext(r.Context())
	event := &models.Event{
		Type:       models.EventTypeSlotDisabled,
		StartsAt:   startsAt,
		OccurredAt: time.Now(),
	}
	if user != nil {
		event.UserID = user.ID
	}
	if err := h.eventRepo.Create(event); err != nil {
		log.Printf("Error creating event: %v", err)
	}

	sendJSON(w, http.StatusOK, map[string]string{"message": "Slot disabled successfully"})
}

// DisableSlotConfirm disables a slot and deletes all associated bookings
func (h *BookingHandler) DisableSlotConfirm(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method not allowed"})
		return
	}

	var req struct {
		StartsAt  string `json:"startsAt"`
		Confirmed bool   `json:"confirmed"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request"})
		return
	}

	if !req.Confirmed {
		sendJSON(w, http.StatusBadRequest, map[string]string{"error": "Confirmation required"})
		return
	}

	startsAt, err := time.Parse(time.RFC3339, req.StartsAt)
	if err != nil {
		sendJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid date format"})
		return
	}

	// Get all bookings for this slot
	bookings, err := h.bookingRepo.GetBySlotTime(startsAt)
	if err != nil {
		log.Printf("Error getting bookings for slot: %v", err)
		sendJSON(w, http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
		return
	}

	// Delete all bookings and refund accesses
	user := middleware.GetUserFromContext(r.Context())
	for _, booking := range bookings {
		// Delete booking
		if err := h.bookingRepo.Delete(booking.ID); err != nil {
			log.Printf("Error deleting booking %d: %v", booking.ID, err)
			continue
		}

		// Refund remaining access
		if err := h.userRepo.IncrementRemainingAccesses(booking.UserID); err != nil {
			log.Printf("Error refunding access for user %s: %v", booking.UserID, err)
		}

		// Create event log
		event := &models.Event{
			UserID:     booking.UserID,
			StartsAt:   booking.StartsAt,
			Type:       models.EventTypeDeleted,
			OccurredAt: time.Now(),
		}
		if err := h.eventRepo.Create(event); err != nil {
			log.Printf("Error creating event: %v", err)
		}

		// Send notification email
		bookingUser, err := h.userRepo.GetByID(booking.UserID)
		if err == nil {
			go func(u *models.User, t time.Time) {
				if err := h.mailer.SendDeleteBookingNotification(u.FirstName, u.LastName.String, t); err != nil {
					log.Printf("Error sending notification: %v", err)
				}
			}(bookingUser, booking.StartsAt)
		}
	}

	// Now disable the slot
	slot, err := h.slotRepo.GetByTime(startsAt)
	if err != nil {
		log.Printf("Error getting slot: %v", err)
		sendJSON(w, http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
		return
	}

	slot.Disabled = true
	if err := h.slotRepo.Update(slot); err != nil {
		log.Printf("Error updating slot: %v", err)
		sendJSON(w, http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
		return
	}

	// Create event log for slot disable
	event := &models.Event{
		Type:       models.EventTypeSlotDisabled,
		StartsAt:   startsAt,
		OccurredAt: time.Now(),
	}
	if user != nil {
		event.UserID = user.ID
	}
	if err := h.eventRepo.Create(event); err != nil {
		log.Printf("Error creating event: %v", err)
	}

	sendJSON(w, http.StatusOK, map[string]interface{}{
		"message":      "Slot disabled and bookings deleted",
		"deletedCount": len(bookings),
	})
}

// EnableSlot allows admin to re-enable a disabled slot
func (h *BookingHandler) EnableSlot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method not allowed"})
		return
	}

	var req struct {
		StartsAt string `json:"startsAt"`
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

	// Get the slot
	slot, err := h.slotRepo.GetByTime(startsAt)
	if err != nil {
		log.Printf("Error getting slot: %v", err)
		sendJSON(w, http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
		return
	}

	// Re-enable the slot
	slot.Disabled = false
	if err := h.slotRepo.Update(slot); err != nil {
		log.Printf("Error updating slot: %v", err)
		sendJSON(w, http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
		return
	}

	// Create event log
	user := middleware.GetUserFromContext(r.Context())
	event := &models.Event{
		Type:       models.EventTypeSlotEnabled,
		StartsAt:   startsAt,
		OccurredAt: time.Now(),
	}
	if user != nil {
		event.UserID = user.ID
	}
	if err := h.eventRepo.Create(event); err != nil {
		log.Printf("Error creating event: %v", err)
	}

	sendJSON(w, http.StatusOK, map[string]string{
		"message": "Slot riabilitato con successo",
	})
}

// CreateBookingForUser allows admin to create a booking for a specific user
func (h *BookingHandler) CreateBookingForUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method not allowed"})
		return
	}

	var req struct {
		UserID       string `json:"userId"`
		StartsAt     string `json:"startsAt"`
		InstructorID string `json:"instructorId"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request"})
		return
	}

	if req.UserID == "" || req.StartsAt == "" || req.InstructorID == "" {
		sendJSON(w, http.StatusBadRequest, map[string]string{"error": "userId, startsAt and instructorId are required"})
		return
	}

	startsAt, err := time.Parse(time.RFC3339, req.StartsAt)
	if err != nil {
		sendJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid date format"})
		return
	}

	// Get the user
	user, err := h.userRepo.GetByID(req.UserID)
	if err != nil {
		if err == sql.ErrNoRows {
			sendJSON(w, http.StatusBadRequest, map[string]string{"error": "User not found"})
			return
		}
		log.Printf("Error getting user: %v", err)
		sendJSON(w, http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
		return
	}

	// Check if user can create booking
	if time.Now().After(user.ExpiresAt) || user.RemainingAccesses <= 0 {
		sendJSON(w, http.StatusBadRequest, map[string]string{"error": "User subscription expired or no remaining accesses"})
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

	// Check slot capacity - check if slot is at capacity based on people_count
	// Note: The original code used slot.Capacity but our Slot struct doesn't have it
	// We'll check against people_count which tracks current bookings

	// Verify instructor exists
	_, err = h.instructorRepo.GetByID(req.InstructorID)
	if err != nil {
		if err == sql.ErrNoRows {
			sendJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid instructor"})
			return
		}
		log.Printf("Error getting instructor: %v", err)
		sendJSON(w, http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
		return
	}

	// Check instructor slot capacity (max 2 people per instructor per slot)
	instructorSlot, err := h.instructorSlotRepo.GetByInstructorAndTime(req.InstructorID, startsAt)
	if err != nil {
		if err == sql.ErrNoRows {
			// Create instructor slot
			instructorSlot = &models.InstructorSlot{
				InstructorID: req.InstructorID,
				StartsAt:     startsAt,
				PeopleCount:  0,
				MaxCapacity:  2,
			}
			if err := h.instructorSlotRepo.Create(instructorSlot); err != nil {
				log.Printf("Error creating instructor slot: %v", err)
				sendJSON(w, http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
				return
			}
		} else {
			log.Printf("Error getting instructor slot: %v", err)
			sendJSON(w, http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
			return
		}
	}

	// Check if instructor slot is full
	if instructorSlot.PeopleCount >= instructorSlot.MaxCapacity {
		sendJSON(w, http.StatusBadRequest, map[string]string{"error": "This instructor's slot is full"})
		return
	}

	// Create booking
	booking := &models.Booking{
		UserID:       user.ID,
		InstructorID: sql.NullString{String: req.InstructorID, Valid: true},
		StartsAt:     startsAt,
		CreatedAt:    time.Now(),
	}

	if err := h.bookingRepo.Create(booking); err != nil {
		log.Printf("Error creating booking: %v", err)
		sendJSON(w, http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
		return
	}

	// Update slot people count
	if err := h.slotRepo.IncrementPeopleCount(startsAt); err != nil {
		log.Printf("Error updating slot: %v", err)
	}

	// Update instructor slot people count
	if err := h.instructorSlotRepo.IncrementPeopleCount(req.InstructorID, startsAt); err != nil {
		log.Printf("Error updating instructor slot: %v", err)
	}

	// Update user's remaining accesses
	user.RemainingAccesses--
	if err := h.userRepo.Update(user); err != nil {
		log.Printf("Error updating user: %v", err)
	}

	// Create event log
	event := &models.Event{
		Type:       models.EventTypeBookingCreated,
		UserID:     user.ID,
		StartsAt:   startsAt,
		OccurredAt: time.Now(),
	}
	if err := h.eventRepo.Create(event); err != nil {
		log.Printf("Error creating event: %v", err)
	}

	// Send notification email (synchronous)
	if err := h.mailer.SendNewBookingNotification(user.FirstName, user.LastName.String, startsAt); err != nil {
		log.Printf("Error sending notification: %v", err)
	}

	sendJSON(w, http.StatusCreated, map[string]interface{}{
		"message":   "Booking created successfully",
		"bookingId": booking.ID,
	})
}
