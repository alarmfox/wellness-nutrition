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
	token := base64.URLEncoding.EncodeToString(b)
	// Sign the token
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

			// Set the CSRF token in a cookie
			// Note: HttpOnly is false so JavaScript can read the token for AJAX requests
			// This is safe for CSRF tokens as they don't grant authentication
			http.SetCookie(w, &http.Cookie{
				Name:     CSRFCookieName,
				Value:    token,
				Path:     "/",
				HttpOnly: false, // JavaScript needs to read this for fetch requests
				Secure:   false, // Set to true in production with HTTPS
				SameSite: http.SameSiteStrictMode,
				MaxAge:   86400, // 24 hours
			})

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
			http.Error(w, "CSRF token missing", http.StatusForbidden)
			return
		}

		// Verify the token signature
		if err := VerifyCSRFToken(token); err != nil {
			log.Printf("Invalid CSRF token for %s %s: %v", r.Method, r.URL.Path, err)
			http.Error(w, "Invalid CSRF token", http.StatusForbidden)
			return
		}

		// Also verify it matches the cookie
		cookie, err := r.Cookie(CSRFCookieName)
		if err != nil {
			log.Printf("CSRF cookie missing for %s %s", r.Method, r.URL.Path)
			http.Error(w, "CSRF cookie missing", http.StatusForbidden)
			return
		}

		if cookie.Value != token {
			log.Printf("CSRF token mismatch for %s %s", r.Method, r.URL.Path)
			http.Error(w, "CSRF token mismatch", http.StatusForbidden)
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

// CSRFExempt creates a middleware that exempts certain paths from CSRF protection
// This is useful for API endpoints that use other authentication methods
func CSRFExempt(exemptPaths ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if path is exempt
			for _, path := range exemptPaths {
				if strings.HasPrefix(r.URL.Path, path) {
					next.ServeHTTP(w, r)
					return
				}
			}

			// Not exempt, apply CSRF protection
			CSRF(next).ServeHTTP(w, r)
		})
	}
}
