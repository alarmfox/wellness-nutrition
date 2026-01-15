package models_test

import (
	"database/sql"
	"testing"
	"time"

	"github.com/alarmfox/wellness-nutrition/app/models"
	"github.com/alarmfox/wellness-nutrition/app/testutil"
	"github.com/google/uuid"
)

func TestBookingRepository_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	testutil.CreateTestSchema(t, db)
	defer testutil.DropTestSchema(t, db)

	// Setup test data
	userRepo := models.NewUserRepository(db)
	instructorRepo := models.NewInstructorRepository(db)
	bookingRepo := models.NewBookingRepository(db)

	// Create test user
	user := &models.User{
		ID:                uuid.New().String(),
		FirstName:         "Test",
		LastName:          "User",
		Email:             "test@example.com",
		Role:              models.RoleUser,
		SubType:           models.SubTypeSingle,
		ExpiresAt:         time.Now().Add(30 * 24 * time.Hour),
		RemainingAccesses: 10,
	}
	if err := userRepo.Create(user); err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Create test instructor
	_, err := db.Exec("INSERT INTO instructors (id, name) VALUES (1, 'Test Instructor')")
	if err != nil {
		t.Fatalf("Failed to create test instructor: %v", err)
	}

	t.Run("Create and Get Booking", func(t *testing.T) {
		testutil.TruncateTables(t, db, "bookings")

		booking := &models.Booking{
			UserID: sql.NullString{
				String: user.ID,
				Valid:  true,
			},
			InstructorID: 1,
			StartsAt:     time.Now().Add(24 * time.Hour).UTC(),
			Type:         models.BookingTypeSimple,
		}

		err := bookingRepo.Create(booking)
		if err != nil {
			t.Fatalf("Failed to create booking: %v", err)
		}

		if booking.ID == 0 {
			t.Error("Booking ID should be set after creation")
		}

		// Get by ID
		retrieved, err := bookingRepo.GetByID(booking.ID)
		if err != nil {
			t.Fatalf("Failed to get booking by ID: %v", err)
		}

		if retrieved.UserID.String != user.ID {
			t.Errorf("Expected user ID %s, got %s", user.ID, retrieved.UserID.String)
		}

		if retrieved.InstructorID != 1 {
			t.Errorf("Expected instructor ID 1, got %d", retrieved.InstructorID)
		}
	})

	t.Run("Get Bookings by User ID", func(t *testing.T) {
		testutil.TruncateTables(t, db, "bookings")

		// Create multiple bookings for the user
		for i := 0; i < 3; i++ {
			booking := &models.Booking{
				UserID: sql.NullString{
					String: user.ID,
					Valid:  true,
				},
				InstructorID: 1,
				StartsAt:     time.Now().Add(time.Duration(i+1) * 24 * time.Hour).UTC(),
				Type:         models.BookingTypeSimple,
			}
			err := bookingRepo.Create(booking)
			if err != nil {
				t.Fatalf("Failed to create booking: %v", err)
			}
		}

		// Get bookings for user
		bookings, err := bookingRepo.GetByUserID(user.ID)
		if err != nil {
			t.Fatalf("Failed to get bookings by user ID: %v", err)
		}

		if len(bookings) != 3 {
			t.Errorf("Expected 3 bookings, got %d", len(bookings))
		}
	})

	t.Run("Get Bookings by Date Range", func(t *testing.T) {
		testutil.TruncateTables(t, db, "bookings")

		now := time.Now().UTC()

		// Create booking in range
		booking1 := &models.Booking{
			UserID: sql.NullString{
				String: user.ID,
				Valid:  true,
			},
			InstructorID: 1,
			StartsAt:     now.Add(2 * 24 * time.Hour),
			Type:         models.BookingTypeSimple,
		}
		err := bookingRepo.Create(booking1)
		if err != nil {
			t.Fatalf("Failed to create booking 1: %v", err)
		}

		// Create booking out of range
		booking2 := &models.Booking{
			UserID: sql.NullString{
				String: user.ID,
				Valid:  true,
			},
			InstructorID: 1,
			StartsAt:     now.Add(10 * 24 * time.Hour),
			Type:         models.BookingTypeSimple,
		}
		err = bookingRepo.Create(booking2)
		if err != nil {
			t.Fatalf("Failed to create booking 2: %v", err)
		}

		// Query with date range
		from := now.Add(1 * 24 * time.Hour)
		to := now.Add(5 * 24 * time.Hour)

		bookings, err := bookingRepo.GetByDateRange(from, to)
		if err != nil {
			t.Fatalf("Failed to get bookings by date range: %v", err)
		}

		if len(bookings) != 1 {
			t.Errorf("Expected 1 booking in range, got %d", len(bookings))
		}

		if len(bookings) > 0 && bookings[0].ID != booking1.ID {
			t.Errorf("Expected booking ID %d, got %d", booking1.ID, bookings[0].ID)
		}
	})

	t.Run("Delete Booking", func(t *testing.T) {
		testutil.TruncateTables(t, db, "bookings")

		booking := &models.Booking{
			UserID: sql.NullString{
				String: user.ID,
				Valid:  true,
			},
			InstructorID: 1,
			StartsAt:     time.Now().Add(24 * time.Hour).UTC(),
			Type:         models.BookingTypeSimple,
		}

		err := bookingRepo.Create(booking)
		if err != nil {
			t.Fatalf("Failed to create booking: %v", err)
		}

		// Delete booking
		err = bookingRepo.Delete(booking.ID)
		if err != nil {
			t.Fatalf("Failed to delete booking: %v", err)
		}

		// Try to retrieve deleted booking
		_, err = bookingRepo.GetByID(booking.ID)
		if err != sql.ErrNoRows {
			t.Errorf("Expected ErrNoRows, got %v", err)
		}
	})

	_ = instructorRepo // Suppress unused warning
}
