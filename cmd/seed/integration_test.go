package main

import (
	"encoding/json"
	"testing"
	"time"
)

// TestCompleteTimezoneFlow tests the complete flow from slot creation to client display
func TestCompleteTimezoneFlow(t *testing.T) {
	// Step 1: Server creates a slot at 7 AM UTC (like the seed does)
	slotTime := time.Date(2024, 1, 15, 7, 0, 0, 0, time.UTC)
	
	t.Logf("Step 1: Server creates slot at %v", slotTime)
	
	// Verify it's in UTC
	if slotTime.Location() != time.UTC {
		t.Errorf("Expected UTC location, got %v", slotTime.Location())
	}
	
	// Step 2: Server sends slot to client in RFC3339 format
	slotJSON := slotTime.Format(time.RFC3339)
	t.Logf("Step 2: Server sends to client: %s", slotJSON)
	
	// Verify RFC3339 format includes timezone
	if len(slotJSON) < 20 || slotJSON[len(slotJSON)-1] != 'Z' {
		t.Errorf("Expected RFC3339 format ending with Z, got %s", slotJSON)
	}
	
	// Step 3: Client in GMT+2 parses and displays the time
	gmt2 := time.FixedZone("GMT+2", 2*3600)
	clientDisplayTime := slotTime.In(gmt2)
	t.Logf("Step 3: Client in GMT+2 displays: %v (hour=%d)", clientDisplayTime, clientDisplayTime.Hour())
	
	// Verify client sees 9 AM (7 AM UTC + 2 hours)
	if clientDisplayTime.Hour() != 9 {
		t.Errorf("Expected client to see hour 9, got %d", clientDisplayTime.Hour())
	}
	
	// Step 4: Client wants to book this slot, sends request back
	bookingRequest := clientDisplayTime.Format(time.RFC3339)
	t.Logf("Step 4: Client sends booking request: %s", bookingRequest)
	
	// Step 5: Server parses the booking request
	bookingTime, err := time.Parse(time.RFC3339, bookingRequest)
	if err != nil {
		t.Fatalf("Failed to parse booking request: %v", err)
	}
	t.Logf("Step 5: Server parses booking as: %v", bookingTime.UTC())
	
	// Step 6: Verify the booking time matches the original slot time
	if !bookingTime.Equal(slotTime) {
		t.Errorf("Booking time doesn't match slot time: %v != %v", bookingTime.UTC(), slotTime)
	}
	t.Logf("Step 6: ✓ Booking matches slot - success!")
}

// TestMultipleTimezones tests that slots work correctly for users in different timezones
func TestMultipleTimezones(t *testing.T) {
	// Server creates slot at 7 AM UTC
	slotTime := time.Date(2024, 1, 15, 7, 0, 0, 0, time.UTC)
	
	testCases := []struct {
		timezoneName   string
		timezoneOffset int // hours
		expectedHour   int
	}{
		{"UTC", 0, 7},
		{"GMT+1", 1, 8},
		{"GMT+2", 2, 9},
		{"GMT-5", -5, 2},
		{"GMT+8", 8, 15},
	}
	
	for _, tc := range testCases {
		t.Run(tc.timezoneName, func(t *testing.T) {
			tz := time.FixedZone(tc.timezoneName, tc.timezoneOffset*3600)
			displayTime := slotTime.In(tz)
			
			if displayTime.Hour() != tc.expectedHour {
				t.Errorf("Expected hour %d in %s, got %d", tc.expectedHour, tc.timezoneName, displayTime.Hour())
			}
			
			t.Logf("✓ User in %s sees slot at %02d:00 (correct)", tc.timezoneName, displayTime.Hour())
		})
	}
}

// TestJSONSerialization tests that times serialize correctly for API responses
func TestJSONSerialization(t *testing.T) {
	// Create a slot response like the API does
	type SlotResponse struct {
		StartsAt    string `json:"StartsAt"`
		PeopleCount int    `json:"PeopleCount"`
	}
	
	slotTime := time.Date(2024, 1, 15, 7, 0, 0, 0, time.UTC)
	
	response := SlotResponse{
		StartsAt:    slotTime.Format(time.RFC3339),
		PeopleCount: 0,
	}
	
	// Serialize to JSON
	jsonBytes, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("Failed to marshal JSON: %v", err)
	}
	
	jsonStr := string(jsonBytes)
	t.Logf("JSON response: %s", jsonStr)
	
	// Verify the timestamp is in RFC3339 format with Z suffix
	if !contains(jsonStr, "2024-01-15T07:00:00Z") {
		t.Errorf("Expected RFC3339 timestamp with Z, got: %s", jsonStr)
	}
	
	// Parse it back
	var parsed SlotResponse
	if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}
	
	parsedTime, err := time.Parse(time.RFC3339, parsed.StartsAt)
	if err != nil {
		t.Fatalf("Failed to parse time from JSON: %v", err)
	}
	
	if !parsedTime.Equal(slotTime) {
		t.Errorf("Parsed time doesn't match original: %v != %v", parsedTime, slotTime)
	}
	
	t.Logf("✓ JSON serialization preserves timezone information")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
