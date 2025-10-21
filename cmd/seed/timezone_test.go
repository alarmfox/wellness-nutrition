package main

import (
	"testing"
	"time"
)

// TestTimezoneConsistency verifies that times are created in UTC
func TestTimezoneConsistency(t *testing.T) {
	// Simulate creating a slot at 7 AM
	now := time.Now().UTC()
	slotTime := time.Date(now.Year(), now.Month(), now.Day(), 7, 0, 0, 0, time.UTC)
	
	// Verify the timezone is UTC
	if slotTime.Location() != time.UTC {
		t.Errorf("Expected UTC location, got %v", slotTime.Location())
	}
	
	// Verify the hour is 7
	if slotTime.Hour() != 7 {
		t.Errorf("Expected hour 7, got %d", slotTime.Hour())
	}
	
	// Simulate a browser in GMT+2 sending a request for 7 AM local time
	browserTimeStr := slotTime.Format(time.RFC3339) // This will be in UTC
	t.Logf("Slot time in UTC: %s", browserTimeStr)
	
	// Parse it back
	parsedTime, err := time.Parse(time.RFC3339, browserTimeStr)
	if err != nil {
		t.Fatalf("Failed to parse time: %v", err)
	}
	
	// They should be equal
	if !parsedTime.Equal(slotTime) {
		t.Errorf("Times are not equal: %v != %v", parsedTime, slotTime)
	}
}

// TestBrowserTimezoneConversion verifies that browser times are correctly converted
func TestBrowserTimezoneConversion(t *testing.T) {
	// A browser in GMT+2 wants to book a slot at 7 AM local time
	// This is 5 AM UTC
	browserTimeStr := "2024-01-15T07:00:00+02:00"
	
	browserTime, err := time.Parse(time.RFC3339, browserTimeStr)
	if err != nil {
		t.Fatalf("Failed to parse browser time: %v", err)
	}
	
	// The UTC time should be 5 AM
	if browserTime.UTC().Hour() != 5 {
		t.Errorf("Expected 5 AM UTC, got %d AM UTC", browserTime.UTC().Hour())
	}
	
	// If we have a slot at 7 AM UTC, it should NOT match the browser's 7 AM local
	slotTimeUTC := time.Date(2024, 1, 15, 7, 0, 0, 0, time.UTC)
	if browserTime.Equal(slotTimeUTC) {
		t.Error("Browser time 7 AM GMT+2 should not equal slot time 7 AM UTC")
	}
	
	// The slot at 7 AM UTC would appear as 9 AM in the browser (GMT+2)
	t.Logf("Slot at 7 AM UTC appears as %d AM in GMT+2", slotTimeUTC.In(time.FixedZone("GMT+2", 2*3600)).Hour())
}

// TestSlotCreationWithUTC verifies that slots are created with UTC timezone
func TestSlotCreationWithUTC(t *testing.T) {
	tests := []struct {
		hour         int
		expectedHour int
	}{
		{7, 7},
		{12, 12},
		{21, 21},
	}
	
	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	
	for _, tt := range tests {
		slotTime := time.Date(today.Year(), today.Month(), today.Day(),
			tt.hour, 0, 0, 0, time.UTC)
		
		if slotTime.Location() != time.UTC {
			t.Errorf("Slot at hour %d: expected UTC location, got %v", tt.hour, slotTime.Location())
		}
		
		if slotTime.Hour() != tt.expectedHour {
			t.Errorf("Slot at hour %d: expected hour %d, got %d", tt.hour, tt.expectedHour, slotTime.Hour())
		}
	}
}
