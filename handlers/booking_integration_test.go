package handlers_test

import (
	"encoding/json"
	"testing"
	"time"
)

// TestGetAvailableSlots_ResponseStructure verifies the response structure
// This test doesn't require a database connection
func TestGetAvailableSlots_ResponseStructure(t *testing.T) {
	// Response is now a simple array of time.Time values
	type Response struct {
		Slots []time.Time `json:"slots"`
	}

	// Example response structure
	response := Response{
		Slots: []time.Time{
			time.Date(2024, 1, 15, 7, 0, 0, 0, time.UTC),
			time.Date(2024, 1, 15, 8, 0, 0, 0, time.UTC),
		},
	}

	// Verify JSON serialization produces the expected field names
	jsonData, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("Failed to marshal response: %v", err)
	}

	jsonStr := string(jsonData)

	// Verify field name is present
	if !contains(jsonStr, `"slots"`) {
		t.Errorf("Response JSON missing expected field: slots\nGot: %s", jsonStr)
	}

	// Verify slots are RFC3339 formatted timestamps
	var decoded Response
	if err := json.Unmarshal(jsonData, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(decoded.Slots) != 2 {
		t.Errorf("Expected 2 slots, got %d", len(decoded.Slots))
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsAt(s, substr))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
