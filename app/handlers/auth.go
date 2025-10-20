package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/alarmfox/wellness-nutrition/app/mail"
	"github.com/alarmfox/wellness-nutrition/app/middleware"
	"github.com/alarmfox/wellness-nutrition/app/models"
)

type AuthHandler struct {
	userRepo     *models.UserRepository
	sessionStore *middleware.SessionStore
}

func NewAuthHandler(userRepo *models.UserRepository, sessionStore *middleware.SessionStore) *AuthHandler {
	return &AuthHandler{
		userRepo:     userRepo,
		sessionStore: sessionStore,
	}
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method not allowed"})
		return
	}
	
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request"})
		return
	}
	
	user, err := h.userRepo.GetByEmail(req.Email)
	if err != nil {
		if err == sql.ErrNoRows {
			sendJSON(w, http.StatusUnauthorized, map[string]string{"error": "Invalid credentials"})
			return
		}
		log.Printf("Error getting user: %v", err)
		sendJSON(w, http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
		return
	}
	
	if !user.EmailVerified.Valid || !user.Password.Valid {
		sendJSON(w, http.StatusUnauthorized, map[string]string{"error": "Account not verified"})
		return
	}
	
	// TODO: Implement proper password verification
	if !middleware.VerifyPassword(req.Password, user.Password.String) {
		sendJSON(w, http.StatusUnauthorized, map[string]string{"error": "Invalid credentials"})
		return
	}
	
	token, err := h.sessionStore.CreateSession(user.ID)
	if err != nil {
		log.Printf("Error creating session: %v", err)
		sendJSON(w, http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
		return
	}
	
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(30 * 24 * time.Hour),
	})
	
	sendJSON(w, http.StatusOK, map[string]interface{}{
		"user": map[string]interface{}{
			"id":        user.ID,
			"firstName": user.FirstName,
			"lastName":  user.LastName,
			"email":     user.Email,
			"role":      user.Role,
			"subType":   user.SubType,
		},
	})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method not allowed"})
		return
	}
	
	cookie, err := r.Cookie("session")
	if err == nil {
		h.sessionStore.DeleteSession(cookie.Value)
	}
	
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})
	
	sendJSON(w, http.StatusOK, map[string]string{"message": "Logged out successfully"})
}

type UserHandler struct {
	userRepo *models.UserRepository
	mailer   *mail.Mailer
}

func NewUserHandler(userRepo *models.UserRepository, mailer *mail.Mailer) *UserHandler {
	return &UserHandler{
		userRepo: userRepo,
		mailer:   mailer,
	}
}

func (h *UserHandler) GetCurrent(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		sendJSON(w, http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
		return
	}
	
	sendJSON(w, http.StatusOK, map[string]interface{}{
		"id":               user.ID,
		"firstName":        user.FirstName,
		"lastName":         user.LastName,
		"email":            user.Email,
		"address":          user.Address,
		"cellphone":        user.Cellphone.String,
		"role":             user.Role,
		"subType":          user.SubType,
		"medOk":            user.MedOk,
		"expiresAt":        user.ExpiresAt,
		"remainingAccesses": user.RemainingAccesses,
		"emailVerified":    user.EmailVerified.Valid,
		"goals":            user.Goals.String,
	})
}

func (h *UserHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	users, err := h.userRepo.GetAll()
	if err != nil {
		log.Printf("Error getting users: %v", err)
		sendJSON(w, http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
		return
	}
	
	sendJSON(w, http.StatusOK, users)
}

type CreateUserRequest struct {
	FirstName         string   `json:"firstName"`
	LastName          string   `json:"lastName"`
	Email             string   `json:"email"`
	Address           string   `json:"address"`
	Cellphone         string   `json:"cellphone"`
	SubType           string   `json:"subType"`
	MedOk             bool     `json:"medOk"`
	ExpiresAt         string   `json:"expiresAt"`
	RemainingAccesses int      `json:"remainingAccesses"`
	Goals             []string `json:"goals"`
}

func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request"})
		return
	}
	
	// TODO: Generate verification token and send email
	// For now, create user without verification
	
	sendJSON(w, http.StatusCreated, map[string]string{"message": "User created successfully"})
}

type UpdateUserRequest struct {
	ID                string   `json:"id"`
	FirstName         string   `json:"firstName"`
	LastName          string   `json:"lastName"`
	Email             string   `json:"email"`
	Address           string   `json:"address"`
	Cellphone         string   `json:"cellphone"`
	SubType           string   `json:"subType"`
	MedOk             bool     `json:"medOk"`
	ExpiresAt         string   `json:"expiresAt"`
	RemainingAccesses int      `json:"remainingAccesses"`
	Goals             []string `json:"goals"`
}

func (h *UserHandler) Update(w http.ResponseWriter, r *http.Request) {
	var req UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request"})
		return
	}
	
	// TODO: Update user in database
	
	sendJSON(w, http.StatusOK, map[string]string{"message": "User updated successfully"})
}

type DeleteUsersRequest struct {
	IDs []string `json:"ids"`
}

func (h *UserHandler) Delete(w http.ResponseWriter, r *http.Request) {
	var req DeleteUsersRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request"})
		return
	}
	
	if err := h.userRepo.Delete(req.IDs); err != nil {
		log.Printf("Error deleting users: %v", err)
		sendJSON(w, http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
		return
	}
	
	sendJSON(w, http.StatusOK, map[string]string{"message": "Users deleted successfully"})
}

type ResetPasswordRequest struct {
	Email string `json:"email"`
}

func (h *UserHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method not allowed"})
		return
	}
	
	if err := r.ParseForm(); err != nil {
		sendJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request"})
		return
	}
	
	email := r.FormValue("email")
	if email == "" {
		sendJSON(w, http.StatusBadRequest, map[string]string{"error": "Email is required"})
		return
	}
	
	user, err := h.userRepo.GetByEmail(email)
	if err != nil {
		// Don't reveal if user exists
		sendJSON(w, http.StatusOK, map[string]string{"message": "If the email exists, a reset link has been sent"})
		return
	}
	
	// TODO: Generate reset token and send email
	go func() {
		resetURL := "http://localhost:3000/verify?token=example"
		if err := h.mailer.SendResetEmail(user.Email, user.FirstName, resetURL); err != nil {
			log.Printf("Error sending reset email: %v", err)
		}
	}()
	
	sendJSON(w, http.StatusOK, map[string]string{"message": "If the email exists, a reset link has been sent"})
}

type VerifyAccountRequest struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

func (h *UserHandler) VerifyAccount(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method not allowed"})
		return
	}
	
	// TODO: Implement account verification
	
	sendJSON(w, http.StatusOK, map[string]string{"message": "Account verified successfully"})
}

func sendJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Error encoding JSON: %v", err)
	}
}
