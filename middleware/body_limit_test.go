package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBodyLimitRejectsOversizedRequest(t *testing.T) {
	handler := BodyLimit(4)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader("abcdef"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected oversized request to fail with 400, got %d", w.Code)
	}
}
