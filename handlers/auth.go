package handlers

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"mime"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/alarmfox/wellness-nutrition/app/crypto"
	"github.com/alarmfox/wellness-nutrition/app/mail"
	"github.com/alarmfox/wellness-nutrition/app/middleware"
	"github.com/alarmfox/wellness-nutrition/app/models"
	"github.com/google/uuid"
)

type AuthHandler struct {
	userRepo     *models.UserRepository
	sessionStore *models.SessionStore
}

func NewAuthHandler(userRepo *models.UserRepository, sessionStore *models.SessionStore) *AuthHandler {
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
	var req LoginRequest

	if isJSONContentType(r.Header.Get("Content-Type")) {
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
		sendJSON(w, http.StatusUnauthorized, map[string]string{"error": "Invalid credentials"})
		return
	}

	// Verify password using centralized crypto
	if !crypto.VerifyPassword(req.Password, user.Password.String) {
		sendJSON(w, http.StatusUnauthorized, map[string]string{"error": "Invalid credentials"})
		return
	}

	token, err := h.sessionStore.CreateSession(user.ID)
	if err != nil {
		log.Printf("Error creating session: %v", err)
		sendJSON(w, http.StatusInternalServerError, map[string]string{"error": "Internal server error"})
		return
	}

	middleware.SetSessionCookie(w, token, time.Now().Add(30*24*time.Hour))

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
	cookie, err := r.Cookie("session")
	if err == nil {
		h.sessionStore.DeleteSession(cookie.Value)
	}

	middleware.ClearSessionCookie(w)
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
		"id":                user.ID,
		"firstName":         user.FirstName,
		"lastName":          user.LastName,
		"email":             user.Email,
		"address":           user.Address,
		"cellphone":         user.Cellphone.String,
		"role":              user.Role,
		"subType":           user.SubType,
		"medOk":             user.MedOk,
		"expiresAt":         user.ExpiresAt,
		"remainingAccesses": user.RemainingAccesses,
		"emailVerified":     user.EmailVerified.Valid,
		"goals":             user.Goals.String,
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
	tokenExpiresAt := time.Now().Add(7 * 24 * time.Hour)
	signedToken, unsignedToken, err := generateSignedToken(tokenExpiresAt)
	if err != nil {
		log.Printf("Error generating token: %v", err)
		sendJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to generate verification token"})
		return
	}

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
		VerificationToken:          sql.NullString{String: unsignedToken, Valid: true},
		VerificationTokenExpiresIn: sql.NullTime{Time: tokenExpiresAt, Valid: true},
		Goals:                      sql.NullString{String: goals, Valid: goals != ""},
	}

	if err := h.userRepo.Create(user); err != nil {
		log.Printf("Error creating user: %v", err)
		sendJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to create user"})
		return
	}

	// Send welcome email with verification link
	verificationURL := fmt.Sprintf("%s/verify?token=%s", getBaseURL(r), url.QueryEscape(signedToken))
	if err := h.mailer.SendWelcomeEmail(user.Email, user.FirstName, verificationURL); err != nil {
		log.Printf("Error sending welcome email: %v", err)
		// Don't fail user creation if email fails, but log it
	}

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
		expiresAt := time.Now().Add(7 * 24 * time.Hour)
		signedToken, unsignedToken, err := generateSignedToken(expiresAt)
		if err != nil {
			log.Printf("Error generating token: %v", err)
			sendJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to generate verification token"})
			return
		}
		user.EmailVerified = sql.NullTime{Valid: false}
		user.VerificationToken = sql.NullString{String: unsignedToken, Valid: true}
		user.VerificationTokenExpiresIn = sql.NullTime{Time: expiresAt, Valid: true}

		// Send new verification email
		verificationURL := fmt.Sprintf("%s/verify?token=%s", getBaseURL(r), url.QueryEscape(signedToken))
		if err := h.mailer.SendWelcomeEmail(user.Email, user.FirstName, verificationURL); err != nil {
			log.Printf("Error sending verification email: %v", err)
			// Don't fail update if email fails, but log it
		}
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

	w.WriteHeader(http.StatusNoContent)
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
	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	signedToken, unsignedToken, err := generateSignedToken(expiresAt)
	if err != nil {
		log.Printf("Error generating token: %v", err)
		sendJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to generate verification token"})
		return
	}
	user.VerificationToken = sql.NullString{String: unsignedToken, Valid: true}
	user.VerificationTokenExpiresIn = sql.NullTime{Time: expiresAt, Valid: true}

	if err := h.userRepo.Update(user); err != nil {
		log.Printf("Error updating user: %v", err)
		sendJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to update user"})
		return
	}

	// Send verification email
	verificationURL := fmt.Sprintf("%s/verify?token=%s", getBaseURL(r), url.QueryEscape(signedToken))
	if err := h.mailer.SendWelcomeEmail(user.Email, user.FirstName, verificationURL); err != nil {
		log.Printf("Error sending verification email: %v", err)
		sendJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to send verification email"})
		return
	}

	sendJSON(w, http.StatusOK, map[string]string{"message": "Verification email sent successfully"})
}

// Helper functions
func generateID() string {
	return uuid.New().String()
}

func generateSignedToken(expiresAt time.Time) (signedToken, unsignedToken string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", err
	}
	unsignedToken = base64.RawURLEncoding.EncodeToString(b)
	// Sign the token with the expiration time
	signedToken = crypto.CreateTimedToken(unsignedToken, expiresAt)
	return signedToken, unsignedToken, nil
}

func getBaseURL(r *http.Request) string {
	if baseURL := strings.TrimRight(os.Getenv("AUTH_URL"), "/"); baseURL != "" {
		return baseURL
	}

	if os.Getenv("ENVIRONMENT") == "production" {
		return ""
	}

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
	var req ResetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request"})
		return
	}

	email := req.Email
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

	// Generate reset token
	expiresAt := time.Now().Add(1 * time.Hour)
	signedToken, unsignedToken, err := generateSignedToken(expiresAt)
	if err != nil {
		log.Printf("Error generating reset token: %v", err)
		sendJSON(w, http.StatusOK, map[string]string{"message": "If the email exists, a reset link has been sent"})
		return
	}

	// Update user with reset token
	user.VerificationToken = sql.NullString{String: unsignedToken, Valid: true}
	user.VerificationTokenExpiresIn = sql.NullTime{Time: expiresAt, Valid: true}

	if err := h.userRepo.Update(user); err != nil {
		log.Printf("Error updating user with reset token: %v", err)
		sendJSON(w, http.StatusOK, map[string]string{"message": "If the email exists, a reset link has been sent"})
		return
	}

	// Send reset email
	resetURL := fmt.Sprintf("%s/reset?token=%s", getBaseURL(r), url.QueryEscape(signedToken))
	if err := h.mailer.SendResetEmail(user.Email, user.FirstName, resetURL); err != nil {
		log.Printf("Error sending reset email: %v", err)
	}

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

	var req VerifyAccountRequest

	if isJSONContentType(r.Header.Get("Content-Type")) {
		// Parse JSON
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			sendJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request"})
			return
		}
	} else {
		// Parse form data (default for HTML form submissions)
		if err := r.ParseForm(); err != nil {
			sendJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request"})
			return
		}
		req.Token = r.FormValue("token")
		req.Password = r.FormValue("password")
	}

	// Validate input
	if req.Token == "" || req.Password == "" {
		sendJSON(w, http.StatusBadRequest, map[string]string{"error": "Token and password are required"})
		return
	}

	// Verify the signed token and extract the unsigned token
	unsignedToken, err := crypto.VerifyTimedToken(req.Token)
	if err != nil {
		if err == crypto.ErrExpiredToken {
			sendJSON(w, http.StatusBadRequest, map[string]string{"error": "Verification token has expired"})
			return
		}
		sendJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid verification token"})
		return
	}

	// Find user by the unsigned verification token
	user, err := h.userRepo.GetByVerificationToken(unsignedToken)
	if err != nil {
		sendJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid or expired verification token"})
		return
	}

	// Hash password using centralized crypto
	hashedPassword, err := crypto.HashPassword(req.Password)
	if err != nil {
		log.Printf("Error hashing password: %v", err)
		sendJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to set password"})
		return
	}

	// Update user
	user.Password = sql.NullString{String: hashedPassword, Valid: true}
	user.EmailVerified = sql.NullTime{Time: time.Now(), Valid: true}
	user.VerificationToken = sql.NullString{Valid: false}
	user.VerificationTokenExpiresIn = sql.NullTime{Valid: false}

	if err := h.userRepo.Update(user); err != nil {
		log.Printf("Error updating user: %v", err)
		sendJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to verify account"})
		return
	}

	sendJSON(w, http.StatusOK, map[string]string{"message": "Account verified successfully"})
}

func sendJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Error encoding JSON: %v", err)
	}
}

func isJSONContentType(contentType string) bool {
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return false
	}
	return mediaType == "application/json"
}
