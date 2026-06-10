package handlers

import "testing"

func TestIsJSONContentTypeAcceptsCharset(t *testing.T) {
	if !isJSONContentType("application/json; charset=utf-8") {
		t.Fatal("expected JSON content type with charset to be accepted")
	}
}

func TestGetBaseURLDoesNotFallbackInProduction(t *testing.T) {
	t.Setenv("AUTH_URL", "")
	t.Setenv("ENVIRONMENT", "production")

	if got := getBaseURL(nil); got != "" {
		t.Fatalf("expected no host fallback in production, got %q", got)
	}
}
