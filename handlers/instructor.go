package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/alarmfox/wellness-nutrition/app/models"
)

type InstructorHandler struct {
	instructorRepo *models.InstructorRepository
	cacheMu        sync.Mutex
	cacheExpiresAt time.Time
	enabledCache   []*models.Instructor
}

func NewInstructorHandler(instructorRepo *models.InstructorRepository) *InstructorHandler {
	return &InstructorHandler{
		instructorRepo: instructorRepo,
	}
}

func (h *InstructorHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	instructors, err := h.getEnabledInstructors()
	if err != nil {
		log.Printf("Error getting instructors: %v", err)
		sendJSON(w, http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
		return
	}

	sendJSON(w, http.StatusOK, instructors)
}

func (h *InstructorHandler) getEnabledInstructors() ([]*models.Instructor, error) {
	now := time.Now()

	h.cacheMu.Lock()
	if now.Before(h.cacheExpiresAt) && h.enabledCache != nil {
		instructors := cloneInstructors(h.enabledCache)
		h.cacheMu.Unlock()
		return instructors, nil
	}
	h.cacheMu.Unlock()

	instructors, err := h.instructorRepo.GetEnabled()
	if err != nil {
		return nil, err
	}

	h.cacheMu.Lock()
	h.enabledCache = cloneInstructors(instructors)
	h.cacheExpiresAt = now.Add(referenceCacheTTL)
	h.cacheMu.Unlock()

	return cloneInstructors(instructors), nil
}

func (h *InstructorHandler) invalidateEnabledCache() {
	h.cacheMu.Lock()
	h.cacheExpiresAt = time.Time{}
	h.enabledCache = nil
	h.cacheMu.Unlock()
}

type CreateInstructorRequest struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	MaxSlots  int    `json:"maxSlots"`
	Enabled   *bool  `json:"enabled"`
}

func (h *InstructorHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateInstructorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request"})
		return
	}

	// Validate required fields
	if req.FirstName == "" {
		sendJSON(w, http.StatusBadRequest, map[string]string{"error": "First name is required"})
		return
	}

	if req.MaxSlots <= 0 {
		req.MaxSlots = 2
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	// Create instructor
	instructor := &models.Instructor{
		FirstName: req.FirstName,
		LastName:  req.LastName,
		MaxSlots:  req.MaxSlots,
		Enabled:   enabled,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := h.instructorRepo.Create(instructor); err != nil {
		log.Printf("Error creating instructor: %v", err)
		sendJSON(w, http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
		return
	}
	h.invalidateEnabledCache()

	sendJSON(w, http.StatusCreated, instructor)
}

type UpdateInstructorRequest struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	MaxSlots  int    `json:"maxSlots"`
	Enabled   *bool  `json:"enabled"`
}

func (h *InstructorHandler) Update(w http.ResponseWriter, r *http.Request) {
	var req UpdateInstructorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request"})
		return
	}
	id := r.PathValue("id")

	idInt, err := strconv.ParseInt(id, 10, 32)
	if err != nil {
		sendJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid ID"})
		return
	}

	// Get existing instructor
	instructor, err := h.instructorRepo.GetByID(idInt)
	if err != nil {
		if err == sql.ErrNoRows {
			sendJSON(w, http.StatusNotFound, map[string]string{"error": "Instructor not found"})
			return
		}
		log.Printf("Error getting instructor: %v", err)
		sendJSON(w, http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
		return
	}

	// Update fields
	instructor.FirstName = req.FirstName
	instructor.LastName = req.LastName
	if req.MaxSlots > 0 {
		instructor.MaxSlots = req.MaxSlots
	}
	if req.Enabled != nil {
		instructor.Enabled = *req.Enabled
	}

	if err := h.instructorRepo.Update(instructor); err != nil {
		log.Printf("Error updating instructor: %v", err)
		sendJSON(w, http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
		return
	}
	h.invalidateEnabledCache()

	sendJSON(w, http.StatusOK, instructor)
}

func (h *InstructorHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	idInt, err := strconv.ParseInt(id, 10, 32)
	if err != nil {
		sendJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid ID"})
		return
	}

	if err := h.instructorRepo.Delete(idInt); err != nil {
		log.Printf("Error deleting instructor: %v", err)
		sendJSON(w, http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
		return
	}
	h.invalidateEnabledCache()

	sendJSON(w, http.StatusOK, map[string]string{"message": "Instructor deleted successfully"})
}
