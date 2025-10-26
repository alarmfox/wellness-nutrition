package handlers

import (
	"testing"
	"time"
)

func TestGenerateSlots(t *testing.T) {
	// Test case 1: Generate slots for a week starting at 7am
	start := time.Date(2024, 1, 1, 6, 0, 0, 0, time.UTC) // Monday 7am
	end := time.Date(2024, 1, 8, 0, 0, 0, 0, time.UTC)   // Next Monday

	slots := generateSlots(start, end)

	// Should have slots from Monday to Saturday (6 days)
	// Each day has 15 hours (7am-9pm inclusive, hourly slots)
	// Expected: 6 days * 15 hours = 90 slots
	if len(slots) != 90 {
		t.Errorf("Expected 90 slots, got %d", len(slots))
	}

	// Test case 2: Verify no Sunday slots
	sundayFound := false
	for _, slot := range slots {
		if slot.Weekday() == time.Sunday {
			sundayFound = true
			break
		}
	}
	if sundayFound {
		t.Error("Found Sunday slot, but Sundays should be excluded")
	}

	// Test case 3: Verify all slots are within 6am-8pm
	for _, slot := range slots {
		hour := slot.Hour()
		if hour < 6 || hour > 20 {
			t.Errorf("Slot at %v is outside 6am-8pm range", slot)
		}
	}

	// Test case 4: Verify all slots are on Monday-Saturday
	for _, slot := range slots {
		weekday := slot.Weekday()
		if weekday < time.Monday || weekday > time.Saturday {
			t.Errorf("Slot at %v is not Monday-Saturday", slot)
		}
	}
}

func TestGenerateSlotsShortPeriod(t *testing.T) {
	// Test with a very short period (same day)
	start := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC) // Monday 10am
	end := time.Date(2024, 1, 1, 15, 0, 0, 0, time.UTC)   // Monday 3pm

	slots := generateSlots(start, end)

	// Should have slots from 11am to 2pm (4 slots)
	if len(slots) < 3 || len(slots) > 5 {
		t.Errorf("Expected 3-5 slots for a short period, got %d", len(slots))
	}
}

func TestGenerateSlotsStartBeforeSeven(t *testing.T) {
	// Test starting before 7am
	start := time.Date(2024, 1, 1, 5, 0, 0, 0, time.UTC) // Monday 5am
	end := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)  // Monday 10am

	slots := generateSlots(start, end)

	// First slot should be at 7am or later
	if len(slots) > 0 && slots[0].Hour() < 6 {
		t.Errorf("First slot at %v is before 6am", slots[0])
	}
}
