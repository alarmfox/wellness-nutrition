package handlers

import (
	"testing"
	"time"
)

func TestGenerateSlots(t *testing.T) {
	loc, err := time.LoadLocation(businessTimeZone)
	if err != nil {
		t.Fatal(err)
	}

	start := time.Date(2024, 1, 1, 0, 0, 0, 0, loc) // Monday
	end := time.Date(2024, 1, 8, 0, 0, 0, 0, loc)   // Next Monday

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
		if slot.In(loc).Weekday() == time.Sunday {
			sundayFound = true
			break
		}
	}
	if sundayFound {
		t.Error("Found Sunday slot, but Sundays should be excluded")
	}

	// Test case 3: Verify all slots are within 7am-9pm Europe/Rome
	for _, slot := range slots {
		hour := slot.In(loc).Hour()
		if hour < 7 || hour > 21 {
			t.Errorf("Slot at %v is outside 7am-9pm Europe/Rome range", slot)
		}
	}

	// Test case 4: Verify all slots are on Monday-Saturday
	for _, slot := range slots {
		weekday := slot.In(loc).Weekday()
		if weekday < time.Monday || weekday > time.Saturday {
			t.Errorf("Slot at %v is not Monday-Saturday", slot)
		}
	}
}

func TestGenerateSlotsShortPeriod(t *testing.T) {
	loc, err := time.LoadLocation(businessTimeZone)
	if err != nil {
		t.Fatal(err)
	}

	// Test with a very short period (same day)
	start := time.Date(2024, 1, 1, 10, 0, 0, 0, loc) // Monday 10am Rome
	end := time.Date(2024, 1, 1, 15, 0, 0, 0, loc)   // Monday 3pm Rome

	slots := generateSlots(start, end)

	// Should have slots from 10am to 2pm (5 slots)
	if len(slots) != 5 {
		t.Errorf("Expected 5 slots for a short period, got %d", len(slots))
	}
}

func TestGenerateSlotsStartBeforeSeven(t *testing.T) {
	loc, err := time.LoadLocation(businessTimeZone)
	if err != nil {
		t.Fatal(err)
	}

	// Test starting before 7am
	start := time.Date(2024, 1, 1, 5, 0, 0, 0, loc) // Monday 5am Rome
	end := time.Date(2024, 1, 1, 10, 0, 0, 0, loc)  // Monday 10am Rome

	slots := generateSlots(start, end)

	// First slot should be at 7am or later
	if len(slots) > 0 && slots[0].In(loc).Hour() < 7 {
		t.Errorf("First slot at %v is before 7am Europe/Rome", slots[0])
	}
}

func TestGenerateSlotsDSTOffsets(t *testing.T) {
	loc, err := time.LoadLocation(businessTimeZone)
	if err != nil {
		t.Fatal(err)
	}

	winterStart := time.Date(2024, 1, 1, 0, 0, 0, 0, loc)
	winterEnd := time.Date(2024, 1, 2, 0, 0, 0, 0, loc)
	winterSlots := generateSlots(winterStart, winterEnd)
	if got, want := winterSlots[0], time.Date(2024, 1, 1, 6, 0, 0, 0, time.UTC); !got.Equal(want) {
		t.Fatalf("winter 07:00 Europe/Rome should be %s, got %s", want, got)
	}

	summerStart := time.Date(2024, 7, 1, 0, 0, 0, 0, loc)
	summerEnd := time.Date(2024, 7, 2, 0, 0, 0, 0, loc)
	summerSlots := generateSlots(summerStart, summerEnd)
	if got, want := summerSlots[0], time.Date(2024, 7, 1, 5, 0, 0, 0, time.UTC); !got.Equal(want) {
		t.Fatalf("summer 07:00 Europe/Rome should be %s, got %s", want, got)
	}
}
