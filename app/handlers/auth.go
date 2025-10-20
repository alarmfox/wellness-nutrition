package handlers

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
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
	
	// Check Content-Type to determine if it's JSON or form data
	contentType := r.Header.Get("Content-Type")
	if contentType == "application/json" {
		// Parse JSON
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			sendJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request format"})
			return
		}
	} else {
		// Parse form data (default for HTML form submissions)
		if err := r.ParseForm(); err != nil {
			sendJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request format"})
			return
		}
		req.Email = r.FormValue("email")
		req.Password = r.FormValue("password")
	}
	
	// Validate input
	if req.Email == "" || req.Password == "" {
		sendJSON(w, http.StatusBadRequest, map[string]string{"error": "Email and password are required"})
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
	
	// Verify password using Argon2
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
		"success": true,
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
	
	// Validate required fields
	if req.FirstName == "" || req.LastName == "" || req.Email == "" || req.Address == "" {
		sendJSON(w, http.StatusBadRequest, map[string]string{"error": "Missing required fields"})
		return
	}
	
	// Check if user already exists
	existing, _ := h.userRepo.GetByEmail(req.Email)
	if existing != nil {
		sendJSON(w, http.StatusConflict, map[string]string{"error": "User with this email already exists"})
		return
	}
	
	// Parse expiration date
	expiresAt, err := time.Parse("2006-01-02", req.ExpiresAt)
	if err != nil {
		sendJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid expiration date format"})
		return
	}
	
	// Generate user ID and verification token
	userID := generateID()
	verificationToken := generateToken()
	
	// Join goals
	goals := ""
	if len(req.Goals) > 0 {
		goals = strings.Join(req.Goals, "-")
	}
	
	// Create user
	user := &models.User{
		ID:                         userID,
		FirstName:                  req.FirstName,
		LastName:                   req.LastName,
		Email:                      req.Email,
		Address:                    req.Address,
		Cellphone:                  sql.NullString{String: req.Cellphone, Valid: req.Cellphone != ""},
		SubType:                    models.SubType(req.SubType),
		MedOk:                      req.MedOk,
		ExpiresAt:                  expiresAt,
		RemainingAccesses:          req.RemainingAccesses,
		Role:                       models.RoleUser,
		VerificationToken:          sql.NullString{String: verificationToken, Valid: true},
		VerificationTokenExpiresIn: sql.NullTime{Time: time.Now().Add(7 * 24 * time.Hour), Valid: true},
		Goals:                      sql.NullString{String: goals, Valid: goals != ""},
	}
	
	if err := h.userRepo.Create(user); err != nil {
		log.Printf("Error creating user: %v", err)
		sendJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to create user"})
		return
	}
	
	// Send welcome email with verification link
	verificationURL := fmt.Sprintf("%s/verify?token=%s", getBaseURL(r), verificationToken)
	go h.mailer.SendWelcomeEmail(user.Email, user.FirstName, verificationURL)
	
	sendJSON(w, http.StatusCreated, map[string]interface{}{
		"message": "User created successfully",
		"userId":  userID,
	})
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
	
	// Get existing user
	user, err := h.userRepo.GetByID(req.ID)
	if err != nil {
		log.Printf("Error getting user: %v", err)
		sendJSON(w, http.StatusNotFound, map[string]string{"error": "User not found"})
		return
	}
	
	// Parse expiration date
	expiresAt, err := time.Parse("2006-01-02", req.ExpiresAt)
	if err != nil {
		sendJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid expiration date format"})
		return
	}
	
	// Join goals
	goals := ""
	if len(req.Goals) > 0 {
		goals = strings.Join(req.Goals, "-")
	}
	
	// Check if email changed
	emailChanged := user.Email != req.Email
	
	// Update user fields
	user.FirstName = req.FirstName
	user.LastName = req.LastName
	user.Email = req.Email
	user.Address = req.Address
	user.Cellphone = sql.NullString{String: req.Cellphone, Valid: req.Cellphone != ""}
	user.SubType = models.SubType(req.SubType)
	user.MedOk = req.MedOk
	user.ExpiresAt = expiresAt
	user.RemainingAccesses = req.RemainingAccesses
	user.Goals = sql.NullString{String: goals, Valid: goals != ""}
	
	// If email changed, reset verification and generate new token
	if emailChanged {
		verificationToken := generateToken()
		user.EmailVerified = sql.NullTime{Valid: false}
		user.VerificationToken = sql.NullString{String: verificationToken, Valid: true}
		user.VerificationTokenExpiresIn = sql.NullTime{Time: time.Now().Add(7 * 24 * time.Hour), Valid: true}
		
		// Send new verification email
		verificationURL := fmt.Sprintf("%s/verify?token=%s", getBaseURL(r), verificationToken)
		go h.mailer.SendWelcomeEmail(user.Email, user.FirstName, verificationURL)
	}
	
	if err := h.userRepo.Update(user); err != nil {
		log.Printf("Error updating user: %v", err)
		sendJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to update user"})
		return
	}
	
	sendJSON(w, http.StatusOK, map[string]interface{}{
		"message":      "User updated successfully",
		"emailChanged": emailChanged,
	})
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

type ResendVerificationRequest struct {
	UserID string `json:"userId"`
}

func (h *UserHandler) ResendVerification(w http.ResponseWriter, r *http.Request) {
	var req ResendVerificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request"})
		return
	}
	
	// Get user
	user, err := h.userRepo.GetByID(req.UserID)
	if err != nil {
		log.Printf("Error getting user: %v", err)
		sendJSON(w, http.StatusNotFound, map[string]string{"error": "User not found"})
		return
	}
	
	// Generate new verification token
	verificationToken := generateToken()
	user.VerificationToken = sql.NullString{String: verificationToken, Valid: true}
	user.VerificationTokenExpiresIn = sql.NullTime{Time: time.Now().Add(7 * 24 * time.Hour), Valid: true}
	
	if err := h.userRepo.Update(user); err != nil {
		log.Printf("Error updating user: %v", err)
		sendJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to update user"})
		return
	}
	
	// Send verification email
	verificationURL := fmt.Sprintf("%s/verify?token=%s", getBaseURL(r), verificationToken)
	go h.mailer.SendWelcomeEmail(user.Email, user.FirstName, verificationURL)
	
	sendJSON(w, http.StatusOK, map[string]string{"message": "Verification email sent successfully"})
}

// Helper functions
func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func generateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

func getBaseURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s", scheme, r.Host)
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
