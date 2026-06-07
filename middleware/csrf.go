package middleware

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"log"
	"net/http"
	"strings"

	"github.com/alarmfox/wellness-nutrition/app/crypto"
)

const (
	CSRFTokenContextKey contextKey = "csrf_token"
	CSRFCookieName                 = "csrf_token"
	CSRFHeaderName                 = "X-CSRF-Token"
)

// GenerateCSRFToken generates a new CSRF token
func GenerateCSRFToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	token := base64.RawURLEncoding.EncodeToString(b)
	return crypto.SignToken(token), nil
}

// VerifyCSRFToken verifies a CSRF token
func VerifyCSRFToken(token string) error {
	_, err := crypto.VerifyToken(token)
	return err
}

// CSRF middleware provides CSRF protection
func CSRF(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip CSRF check for GET, HEAD, OPTIONS, TRACE methods (safe methods)
		if r.Method == "GET" || r.Method == "HEAD" || r.Method == "OPTIONS" || r.Method == "TRACE" {
			// Generate a new token for the response (or retrieve existing one)
			token, err := getOrCreateCSRFToken(r)
			if err != nil {
				log.Printf("Error creating CSRF token: %v", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}

			setCSRFCookie(w, token)

			// Add token to context so templates can access it
			ctx := context.WithValue(r.Context(), CSRFTokenContextKey, token)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		// For unsafe methods (POST, PUT, DELETE, PATCH), verify CSRF token
		// Get token from header first, then fall back to form value
		token := r.Header.Get(CSRFHeaderName)
		if token == "" {
			// Try to get from form value
			if err := r.ParseForm(); err == nil {
				token = r.FormValue("csrf_token")
			}
		}

		if token == "" {
			log.Printf("CSRF token missing for %s %s", r.Method, r.URL.Path)
			writeCSRFError(w, r, "CSRF token missing")
			return
		}

		// Verify the token signature
		if err := VerifyCSRFToken(token); err != nil {
			log.Printf("Invalid CSRF token for %s %s: %v", r.Method, r.URL.Path, err)
			writeCSRFError(w, r, "Invalid CSRF token")
			return
		}

		// Also verify it matches the cookie
		cookie, err := r.Cookie(CSRFCookieName)
		if err != nil {
			log.Printf("CSRF cookie missing for %s %s", r.Method, r.URL.Path)
			writeCSRFError(w, r, "CSRF cookie missing")
			return
		}

		if cookie.Value != token {
			log.Printf("CSRF token mismatch for %s %s", r.Method, r.URL.Path)
			writeCSRFError(w, r, "CSRF token mismatch")
			return
		}

		// Token is valid, proceed with request
		ctx := context.WithValue(r.Context(), CSRFTokenContextKey, token)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetCSRFToken retrieves the CSRF token from the context
func GetCSRFToken(ctx context.Context) string {
	token, ok := ctx.Value(CSRFTokenContextKey).(string)
	if !ok {
		return ""
	}
	return token
}

// getOrCreateCSRFToken gets existing CSRF token from cookie or creates a new one
func getOrCreateCSRFToken(r *http.Request) (string, error) {
	// Try to get existing token from cookie
	cookie, err := r.Cookie(CSRFCookieName)
	if err == nil && cookie.Value != "" {
		// Verify the existing token is valid
		if err := VerifyCSRFToken(cookie.Value); err == nil {
			return cookie.Value, nil
		}
	}

	// Generate new token if none exists or existing is invalid
	return GenerateCSRFToken()
}

func writeCSRFError(w http.ResponseWriter, r *http.Request, message string) {
	if isAPIRequest(r) {
		sendJSON(w, http.StatusForbidden, map[string]string{"error": message})
		return
	}

	http.Error(w, message, http.StatusForbidden)
}

func isAPIRequest(r *http.Request) bool {
	return strings.HasPrefix(r.URL.Path, "/api/")
}
