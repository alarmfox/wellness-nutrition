package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/alarmfox/wellness-nutrition/app/models"
)

type SurveyHandler struct {
	questionRepo *models.QuestionRepository
}

func NewSurveyHandler(questionRepo *models.QuestionRepository) *SurveyHandler {
	return &SurveyHandler{
		questionRepo: questionRepo,
	}
}

// SubmitSurvey handles survey submission
func (h *SurveyHandler) SubmitSurvey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		log.Printf("Error parsing form: %v", err)
		return
	}

	// Process each rating
	scores := make(map[int][5]int)
	for k, v := range r.Form {
		// Parse rating-ID format
		parts := strings.Split(k, "-")
		if len(parts) != 2 {
			continue
		}

		id, err := strconv.Atoi(parts[1])
		if err != nil {
			log.Printf("Error parsing question ID: %v", err)
			continue
		}

		if _, exists := scores[id]; !exists {
			scores[id] = [5]int{0, 0, 0, 0, 0}
		}

		star, err := strconv.Atoi(v[0])
		if err != nil || star < 1 || star > 5 {
			log.Printf("Invalid star rating: %v", v[0])
			continue
		}

		stars := scores[id]
		stars[star-1] = 1
		scores[id] = stars
	}

	// Update results for each question
	for id, stars := range scores {
		if err := h.questionRepo.UpdateResults(id, stars); err != nil {
			log.Printf("Error updating results for question %d: %v", id, err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// GetAllQuestions returns all questions (for admin)
func (h *SurveyHandler) GetAllQuestions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	questions, err := h.questionRepo.GetAll()
	if err != nil {
		log.Printf("Error getting questions: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(questions)
}

// CreateQuestion creates a new question (admin only)
func (h *SurveyHandler) CreateQuestion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	var q models.Question
	if err := json.NewDecoder(r.Body).Decode(&q); err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		log.Printf("Error decoding question: %v", err)
		return
	}

	if err := h.questionRepo.Create(&q); err != nil {
		log.Printf("Error creating question: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(q)
}

// UpdateQuestion updates an existing question (admin only)
func (h *SurveyHandler) UpdateQuestion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	var q models.Question
	if err := json.NewDecoder(r.Body).Decode(&q); err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		log.Printf("Error decoding question: %v", err)
		return
	}

	if err := h.questionRepo.Update(&q); err != nil {
		log.Printf("Error updating question: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(q)
}

// DeleteQuestion deletes a question (admin only)
func (h *SurveyHandler) DeleteQuestion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ID int `json:"id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		log.Printf("Error decoding request: %v", err)
		return
	}

	if err := h.questionRepo.Delete(req.ID); err != nil {
		log.Printf("Error deleting question: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// GetResults returns survey results (admin only)
func (h *SurveyHandler) GetResults(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	questions, err := h.questionRepo.GetResults()
	if err != nil {
		log.Printf("Error getting results: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(questions)
}
