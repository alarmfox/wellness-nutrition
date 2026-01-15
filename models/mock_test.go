package models_test

import (
	"database/sql"
	"testing"
	"time"

	"github.com/alarmfox/wellness-nutrition/app/models"
	"github.com/alarmfox/wellness-nutrition/app/testutil"
	"github.com/google/uuid"
)

func TestMockUserRepository(t *testing.T) {
	repo := testutil.NewMockUserRepository()

	t.Run("Create and Get User", func(t *testing.T) {
		repo.Reset()

		user := &models.User{
			ID:                uuid.New().String(),
			FirstName:         "John",
			LastName:          "Doe",
			Email:             "john@example.com",
			Role:              models.RoleUser,
			SubType:           models.SubTypeSingle,
			ExpiresAt:         time.Now().Add(30 * 24 * time.Hour),
			RemainingAccesses: 10,
		}

		err := repo.Create(user)
		if err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}

		// Get by ID
		retrieved, err := repo.GetByID(user.ID)
		if err != nil {
			t.Fatalf("Failed to get user by ID: %v", err)
		}

		if retrieved.Email != user.Email {
			t.Errorf("Expected email %s, got %s", user.Email, retrieved.Email)
		}

		// Get by Email
		retrieved, err = repo.GetByEmail(user.Email)
		if err != nil {
			t.Fatalf("Failed to get user by email: %v", err)
		}

		if retrieved.ID != user.ID {
			t.Errorf("Expected ID %s, got %s", user.ID, retrieved.ID)
		}
	})

	t.Run("Update User", func(t *testing.T) {
		repo.Reset()

		user := &models.User{
			ID:                uuid.New().String(),
			FirstName:         "Jane",
			LastName:          "Doe",
			Email:             "jane@example.com",
			Role:              models.RoleUser,
			SubType:           models.SubTypeSingle,
			ExpiresAt:         time.Now().Add(30 * 24 * time.Hour),
			RemainingAccesses: 5,
		}

		repo.Create(user)

		// Update user
		user.FirstName = "Janet"
		user.RemainingAccesses = 10

		err := repo.Update(user)
		if err != nil {
			t.Fatalf("Failed to update user: %v", err)
		}

		retrieved, err := repo.GetByID(user.ID)
		if err != nil {
			t.Fatalf("Failed to get user: %v", err)
		}

		if retrieved.FirstName != "Janet" {
			t.Errorf("Expected first name Janet, got %s", retrieved.FirstName)
		}
	})

	t.Run("Increment and Decrement Accesses", func(t *testing.T) {
		repo.Reset()

		user := &models.User{
			ID:                uuid.New().String(),
			FirstName:         "Test",
			LastName:          "User",
			Email:             "test@example.com",
			Role:              models.RoleUser,
			SubType:           models.SubTypeSingle,
			ExpiresAt:         time.Now().Add(30 * 24 * time.Hour),
			RemainingAccesses: 5,
		}

		repo.Create(user)

		// Decrement
		err := repo.DecrementAccesses(user.ID)
		if err != nil {
			t.Fatalf("Failed to decrement: %v", err)
		}

		retrieved, err := repo.GetByID(user.ID)
		if err != nil {
			t.Fatalf("Failed to get user: %v", err)
		}

		if retrieved.RemainingAccesses != 4 {
			t.Errorf("Expected 4 accesses, got %d", retrieved.RemainingAccesses)
		}

		// Increment
		err = repo.IncrementAccesses(user.ID)
		if err != nil {
			t.Fatalf("Failed to increment: %v", err)
		}

		retrieved, err = repo.GetByID(user.ID)
		if err != nil {
			t.Fatalf("Failed to get user: %v", err)
		}

		if retrieved.RemainingAccesses != 5 {
			t.Errorf("Expected 5 accesses, got %d", retrieved.RemainingAccesses)
		}
	})

	t.Run("Delete User", func(t *testing.T) {
		repo.Reset()

		user := &models.User{
			ID:    uuid.New().String(),
			Email: "delete@example.com",
			Role:  models.RoleUser,
		}

		repo.Create(user)

		err := repo.Delete([]string{user.ID})
		if err != nil {
			t.Fatalf("Failed to delete user: %v", err)
		}

		_, err = repo.GetByID(user.ID)
		if err != sql.ErrNoRows {
			t.Errorf("Expected ErrNoRows, got %v", err)
		}
	})

	t.Run("Not Found", func(t *testing.T) {
		repo.Reset()

		_, err := repo.GetByID("non-existent")
		if err != sql.ErrNoRows {
			t.Errorf("Expected ErrNoRows, got %v", err)
		}

		_, err = repo.GetByEmail("nonexistent@example.com")
		if err != sql.ErrNoRows {
			t.Errorf("Expected ErrNoRows, got %v", err)
		}
	})
}

func TestMockBookingRepository(t *testing.T) {
	repo := testutil.NewMockBookingRepository()

	t.Run("Create and Get Booking", func(t *testing.T) {
		repo.Reset()

		booking := &models.Booking{
			UserID: sql.NullString{
				String: "user-123",
				Valid:  true,
			},
			InstructorID: 1,
			StartsAt:     time.Now().Add(24 * time.Hour),
			Type:         models.BookingTypeSimple,
		}

		err := repo.Create(booking)
		if err != nil {
			t.Fatalf("Failed to create booking: %v", err)
		}

		if booking.ID == 0 {
			t.Error("Booking ID should be set after creation")
		}

		retrieved, err := repo.GetByID(booking.ID)
		if err != nil {
			t.Fatalf("Failed to get booking: %v", err)
		}

		if retrieved.UserID.String != "user-123" {
			t.Errorf("Expected user ID user-123, got %s", retrieved.UserID.String)
		}
	})

	t.Run("Get Bookings by User ID", func(t *testing.T) {
		repo.Reset()

		userID := "user-456"

		// Create bookings
		for i := 0; i < 3; i++ {
			booking := &models.Booking{
				UserID: sql.NullString{
					String: userID,
					Valid:  true,
				},
				InstructorID: 1,
				StartsAt:     time.Now().Add(time.Duration(i+1) * 24 * time.Hour),
				Type:         models.BookingTypeSimple,
			}
			repo.Create(booking)
		}

		bookings, err := repo.GetByUserID(userID)
		if err != nil {
			t.Fatalf("Failed to get bookings: %v", err)
		}

		if len(bookings) != 3 {
			t.Errorf("Expected 3 bookings, got %d", len(bookings))
		}
	})

	t.Run("Delete Booking", func(t *testing.T) {
		repo.Reset()

		booking := &models.Booking{
			UserID: sql.NullString{
				String: "user-789",
				Valid:  true,
			},
			InstructorID: 1,
			StartsAt:     time.Now().Add(24 * time.Hour),
			Type:         models.BookingTypeSimple,
		}

		repo.Create(booking)

		err := repo.Delete(booking.ID)
		if err != nil {
			t.Fatalf("Failed to delete booking: %v", err)
		}

		_, err = repo.GetByID(booking.ID)
		if err != sql.ErrNoRows {
			t.Errorf("Expected ErrNoRows, got %v", err)
		}
	})

	t.Run("Get Bookings by Date Range", func(t *testing.T) {
		repo.Reset()

		now := time.Now()
		
		// Create booking in range
		booking1 := &models.Booking{
			UserID: sql.NullString{
				String: "user-111",
				Valid:  true,
			},
			InstructorID: 1,
			StartsAt:     now.Add(2 * 24 * time.Hour),
			Type:         models.BookingTypeSimple,
		}
		repo.Create(booking1)

		// Create booking out of range
		booking2 := &models.Booking{
			UserID: sql.NullString{
				String: "user-111",
				Valid:  true,
			},
			InstructorID: 1,
			StartsAt:     now.Add(10 * 24 * time.Hour),
			Type:         models.BookingTypeSimple,
		}
		repo.Create(booking2)

		from := now.Add(1 * 24 * time.Hour)
		to := now.Add(5 * 24 * time.Hour)
		
		bookings, err := repo.GetByDateRange(from, to)
		if err != nil {
			t.Fatalf("Failed to get bookings by date range: %v", err)
		}

		if len(bookings) != 1 {
			t.Errorf("Expected 1 booking in range, got %d", len(bookings))
		}
	})
}
