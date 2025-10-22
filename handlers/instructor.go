package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/alarmfox/wellness-nutrition/app/models"
	"github.com/google/uuid"
	"golang.org/x/crypto/argon2"
)

type InstructorHandler struct {
	instructorRepo *models.InstructorRepository
}

func NewInstructorHandler(instructorRepo *models.InstructorRepository) *InstructorHandler {
	return &InstructorHandler{
		instructorRepo: instructorRepo,
	}
}

func (h *InstructorHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	instructors, err := h.instructorRepo.GetAll()
	if err != nil {
		log.Printf("Error getting instructors: %v", err)
		sendJSON(w, http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
		return
	}

	sendJSON(w, http.StatusOK, instructors)
}

type CreateInstructorRequest struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Email     string `json:"email"`
	Password  string `json:"password"`
}

func (h *InstructorHandler) Create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method not allowed"})
		return
	}

	var req CreateInstructorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request"})
		return
	}

	// Validate required fields
	if req.FirstName == "" || req.LastName == "" || req.Email == "" {
		sendJSON(w, http.StatusBadRequest, map[string]string{"error": "First name, last name, and email are required"})
		return
	}

	// Check if instructor with email already exists
	existing, err := h.instructorRepo.GetByEmail(req.Email)
	if err == nil && existing != nil {
		sendJSON(w, http.StatusConflict, map[string]string{"error": "Instructor with this email already exists"})
		return
	}

	// Create instructor
	instructor := &models.Instructor{
		ID:        uuid.New().String(),
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Email:     req.Email,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Hash password if provided
	if req.Password != "" {
		hash := argon2.IDKey([]byte(req.Password), []byte("salt"), 1, 64*1024, 4, 32)
		instructor.Password = sql.NullString{String: string(hash), Valid: true}
	}

	if err := h.instructorRepo.Create(instructor); err != nil {
		log.Printf("Error creating instructor: %v", err)
		sendJSON(w, http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
		return
	}

	sendJSON(w, http.StatusCreated, instructor)
}

type UpdateInstructorRequest struct {
	ID        string `json:"id"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Email     string `json:"email"`
	Password  string `json:"password,omitempty"`
}

func (h *InstructorHandler) Update(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method not allowed"})
		return
	}

	var req UpdateInstructorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request"})
		return
	}

	// Get existing instructor
	instructor, err := h.instructorRepo.GetByID(req.ID)
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
	instructor.Email = req.Email

	// Update password if provided
	if req.Password != "" {
		hash := argon2.IDKey([]byte(req.Password), []byte("salt"), 1, 64*1024, 4, 32)
		instructor.Password = sql.NullString{String: string(hash), Valid: true}
	}

	if err := h.instructorRepo.Update(instructor); err != nil {
		log.Printf("Error updating instructor: %v", err)
		sendJSON(w, http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
		return
	}

	sendJSON(w, http.StatusOK, instructor)
}

type DeleteInstructorRequest struct {
	ID string `json:"id"`
}

func (h *InstructorHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method not allowed"})
		return
	}

	var req DeleteInstructorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request"})
		return
	}

	if err := h.instructorRepo.Delete(req.ID); err != nil {
		log.Printf("Error deleting instructor: %v", err)
		sendJSON(w, http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
		return
	}

	sendJSON(w, http.StatusOK, map[string]string{"message": "Instructor deleted successfully"})
}
