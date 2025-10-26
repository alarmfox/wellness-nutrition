package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alarmfox/wellness-nutrition/app/crypto"
)

func TestCSRFMiddleware_SafeMethods(t *testing.T) {
	// Initialize crypto secret key for testing
	if err := crypto.InitializeSecretKey("test-secret-key"); err != nil {
		t.Fatalf("Failed to initialize secret key: %v", err)
	}

	safeMethods := []string{"GET", "HEAD", "OPTIONS", "TRACE"}

	for _, method := range safeMethods {
		t.Run(method, func(t *testing.T) {
			// Create a test handler
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("OK"))
			})

			// Wrap with CSRF middleware
			csrfHandler := CSRF(handler)

			// Create request
			req := httptest.NewRequest(method, "/test", nil)
			w := httptest.NewRecorder()

			// Execute request
			csrfHandler.ServeHTTP(w, req)

			// Check response
			if w.Code != http.StatusOK {
				t.Errorf("Expected status OK for %s request, got %d", method, w.Code)
			}

			// Check that CSRF cookie was set
			cookies := w.Result().Cookies()
			var csrfCookie *http.Cookie
			for _, c := range cookies {
				if c.Name == CSRFCookieName {
					csrfCookie = c
					break
				}
			}

			if csrfCookie == nil {
				t.Errorf("CSRF cookie not set for %s request", method)
			}
		})
	}
}

func TestCSRFMiddleware_UnsafeMethodsWithoutToken(t *testing.T) {
	// Initialize crypto secret key for testing
	if err := crypto.InitializeSecretKey("test-secret-key"); err != nil {
		t.Fatalf("Failed to initialize secret key: %v", err)
	}

	unsafeMethods := []string{"POST", "PUT", "DELETE", "PATCH"}

	for _, method := range unsafeMethods {
		t.Run(method, func(t *testing.T) {
			// Create a test handler
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			// Wrap with CSRF middleware
			csrfHandler := CSRF(handler)

			// Create request without CSRF token
			req := httptest.NewRequest(method, "/test", nil)
			w := httptest.NewRecorder()

			// Execute request
			csrfHandler.ServeHTTP(w, req)

			// Should be forbidden
			if w.Code != http.StatusForbidden {
				t.Errorf("Expected status Forbidden for %s request without CSRF token, got %d", method, w.Code)
			}
		})
	}
}

func TestCSRFMiddleware_UnsafeMethodsWithValidToken(t *testing.T) {
	// Initialize crypto secret key for testing
	if err := crypto.InitializeSecretKey("test-secret-key"); err != nil {
		t.Fatalf("Failed to initialize secret key: %v", err)
	}

	unsafeMethods := []string{"POST", "PUT", "DELETE", "PATCH"}

	for _, method := range unsafeMethods {
		t.Run(method, func(t *testing.T) {
			// Generate a CSRF token
			token, err := GenerateCSRFToken()
			if err != nil {
				t.Fatalf("Failed to generate CSRF token: %v", err)
			}

			// Create a test handler
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("OK"))
			})

			// Wrap with CSRF middleware
			csrfHandler := CSRF(handler)

			// Create request with CSRF token in header
			req := httptest.NewRequest(method, "/test", nil)
			req.Header.Set(CSRFHeaderName, token)
			req.AddCookie(&http.Cookie{
				Name:  CSRFCookieName,
				Value: token,
			})
			w := httptest.NewRecorder()

			// Execute request
			csrfHandler.ServeHTTP(w, req)

			// Should be OK
			if w.Code != http.StatusOK {
				t.Errorf("Expected status OK for %s request with valid CSRF token, got %d", method, w.Code)
			}
		})
	}
}

func TestCSRFMiddleware_UnsafeMethodsWithInvalidToken(t *testing.T) {
	// Initialize crypto secret key for testing
	if err := crypto.InitializeSecretKey("test-secret-key"); err != nil {
		t.Fatalf("Failed to initialize secret key: %v", err)
	}

	// Create a test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Wrap with CSRF middleware
	csrfHandler := CSRF(handler)

	// Create request with invalid CSRF token
	req := httptest.NewRequest("POST", "/test", nil)
	req.Header.Set(CSRFHeaderName, "invalid-token")
	req.AddCookie(&http.Cookie{
		Name:  CSRFCookieName,
		Value: "invalid-token",
	})
	w := httptest.NewRecorder()

	// Execute request
	csrfHandler.ServeHTTP(w, req)

	// Should be forbidden
	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status Forbidden for POST request with invalid CSRF token, got %d", w.Code)
	}
}

func TestCSRFMiddleware_TokenMismatch(t *testing.T) {
	// Initialize crypto secret key for testing
	if err := crypto.InitializeSecretKey("test-secret-key"); err != nil {
		t.Fatalf("Failed to initialize secret key: %v", err)
	}

	// Generate two different tokens
	token1, err := GenerateCSRFToken()
	if err != nil {
		t.Fatalf("Failed to generate CSRF token: %v", err)
	}
	token2, err := GenerateCSRFToken()
	if err != nil {
		t.Fatalf("Failed to generate CSRF token: %v", err)
	}

	// Create a test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Wrap with CSRF middleware
	csrfHandler := CSRF(handler)

	// Create request with mismatched tokens (header vs cookie)
	req := httptest.NewRequest("POST", "/test", nil)
	req.Header.Set(CSRFHeaderName, token1)
	req.AddCookie(&http.Cookie{
		Name:  CSRFCookieName,
		Value: token2,
	})
	w := httptest.NewRecorder()

	// Execute request
	csrfHandler.ServeHTTP(w, req)

	// Should be forbidden due to mismatch
	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status Forbidden for mismatched CSRF tokens, got %d", w.Code)
	}
}

func TestCSRFMiddleware_FormValue(t *testing.T) {
	// Initialize crypto secret key for testing
	if err := crypto.InitializeSecretKey("test-secret-key"); err != nil {
		t.Fatalf("Failed to initialize secret key: %v", err)
	}

	// Generate a CSRF token
	token, err := GenerateCSRFToken()
	if err != nil {
		t.Fatalf("Failed to generate CSRF token: %v", err)
	}

	// Create a test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Wrap with CSRF middleware
	csrfHandler := CSRF(handler)

	// Create POST request with CSRF token in form
	formData := "csrf_token=" + token
	req := httptest.NewRequest("POST", "/test", strings.NewReader(formData))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{
		Name:  CSRFCookieName,
		Value: token,
	})
	w := httptest.NewRecorder()

	// Execute request
	csrfHandler.ServeHTTP(w, req)

	// Should be OK
	if w.Code != http.StatusOK {
		t.Errorf("Expected status OK for POST request with CSRF token in form, got %d", w.Code)
	}
}

func TestGetCSRFToken(t *testing.T) {
	// Initialize crypto secret key for testing
	if err := crypto.InitializeSecretKey("test-secret-key"); err != nil {
		t.Fatalf("Failed to initialize secret key: %v", err)
	}

	// Create a handler that uses GetCSRFToken
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := GetCSRFToken(r.Context())
		if token == "" {
			t.Error("Expected CSRF token in context, got empty string")
		}
		w.WriteHeader(http.StatusOK)
	})

	// Wrap with CSRF middleware
	csrfHandler := CSRF(handler)

	// Create GET request
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	// Execute request
	csrfHandler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status OK, got %d", w.Code)
	}
}
