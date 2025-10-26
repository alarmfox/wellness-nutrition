package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/alarmfox/wellness-nutrition/app/models"
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

	// Create instructor
	instructor := &models.Instructor{
		FirstName: req.FirstName,
		LastName:  req.LastName,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := h.instructorRepo.Create(instructor); err != nil {
		log.Printf("Error creating instructor: %v", err)
		sendJSON(w, http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
		return
	}

	sendJSON(w, http.StatusCreated, instructor)
}

type UpdateInstructorRequest struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
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

	if err := h.instructorRepo.Update(instructor); err != nil {
		log.Printf("Error updating instructor: %v", err)
		sendJSON(w, http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
		return
	}

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

	sendJSON(w, http.StatusOK, map[string]string{"message": "Instructor deleted successfully"})
}
