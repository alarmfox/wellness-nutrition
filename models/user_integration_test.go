package models_test

import (
	"database/sql"
	"testing"
	"time"

	"github.com/alarmfox/wellness-nutrition/app/models"
	"github.com/alarmfox/wellness-nutrition/app/testutil"
	"github.com/google/uuid"
)

func TestUserRepository_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db := testutil.SetupTestDB(t)
	defer testutil.CleanupTestDB(t, db)

	testutil.CreateTestSchema(t, db)
	defer testutil.DropTestSchema(t, db)

	repo := models.NewUserRepository(db)

	t.Run("Create and Get User", func(t *testing.T) {
		testutil.TruncateTables(t, db, "users")

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
		testutil.TruncateTables(t, db, "users")

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

		err := repo.Create(user)
		if err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}

		// Update user
		user.FirstName = "Janet"
		user.RemainingAccesses = 10

		err = repo.Update(user)
		if err != nil {
			t.Fatalf("Failed to update user: %v", err)
		}

		// Retrieve updated user
		retrieved, err := repo.GetByID(user.ID)
		if err != nil {
			t.Fatalf("Failed to get user by ID: %v", err)
		}

		if retrieved.FirstName != "Janet" {
			t.Errorf("Expected first name Janet, got %s", retrieved.FirstName)
		}

		if retrieved.RemainingAccesses != 10 {
			t.Errorf("Expected 10 remaining accesses, got %d", retrieved.RemainingAccesses)
		}
	})

	t.Run("Increment and Decrement Accesses", func(t *testing.T) {
		testutil.TruncateTables(t, db, "users")

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

		err := repo.Create(user)
		if err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}

		// Decrement
		err = repo.DecrementAccesses(user.ID)
		if err != nil {
			t.Fatalf("Failed to decrement accesses: %v", err)
		}

		retrieved, err := repo.GetByID(user.ID)
		if err != nil {
			t.Fatalf("Failed to get user: %v", err)
		}

		if retrieved.RemainingAccesses != 4 {
			t.Errorf("Expected 4 remaining accesses, got %d", retrieved.RemainingAccesses)
		}

		// Increment
		err = repo.IncrementAccesses(user.ID)
		if err != nil {
			t.Fatalf("Failed to increment accesses: %v", err)
		}

		retrieved, err = repo.GetByID(user.ID)
		if err != nil {
			t.Fatalf("Failed to get user: %v", err)
		}

		if retrieved.RemainingAccesses != 5 {
			t.Errorf("Expected 5 remaining accesses, got %d", retrieved.RemainingAccesses)
		}
	})

	t.Run("Delete User", func(t *testing.T) {
		testutil.TruncateTables(t, db, "users")

		user := &models.User{
			ID:                uuid.New().String(),
			FirstName:         "Delete",
			LastName:          "Me",
			Email:             "delete@example.com",
			Role:              models.RoleUser,
			SubType:           models.SubTypeSingle,
			ExpiresAt:         time.Now().Add(30 * 24 * time.Hour),
			RemainingAccesses: 0,
		}

		err := repo.Create(user)
		if err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}

		// Delete user
		err = repo.Delete([]string{user.ID})
		if err != nil {
			t.Fatalf("Failed to delete user: %v", err)
		}

		// Try to retrieve deleted user
		_, err = repo.GetByID(user.ID)
		if err != sql.ErrNoRows {
			t.Errorf("Expected ErrNoRows, got %v", err)
		}
	})

	t.Run("Get All Users", func(t *testing.T) {
		testutil.TruncateTables(t, db, "users")

		// Create multiple users
		for i := 0; i < 3; i++ {
			user := &models.User{
				ID:                uuid.New().String(),
				FirstName:         "User",
				LastName:          string(rune('A' + i)),
				Email:             "user" + string(rune('a'+i)) + "@example.com",
				Role:              models.RoleUser,
				SubType:           models.SubTypeSingle,
				ExpiresAt:         time.Now().Add(30 * 24 * time.Hour),
				RemainingAccesses: 10,
			}
			err := repo.Create(user)
			if err != nil {
				t.Fatalf("Failed to create user: %v", err)
			}
		}

		// Create an admin user (should not be included in GetAll)
		admin := &models.User{
			ID:                uuid.New().String(),
			FirstName:         "Admin",
			LastName:          "User",
			Email:             "admin@example.com",
			Role:              models.RoleAdmin,
			SubType:           models.SubTypeSingle,
			ExpiresAt:         time.Now().Add(30 * 24 * time.Hour),
			RemainingAccesses: 0,
		}
		err := repo.Create(admin)
		if err != nil {
			t.Fatalf("Failed to create admin: %v", err)
		}

		// Get all regular users
		users, err := repo.GetAll()
		if err != nil {
			t.Fatalf("Failed to get all users: %v", err)
		}

		if len(users) != 3 {
			t.Errorf("Expected 3 users, got %d", len(users))
		}
	})
}
