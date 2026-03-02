# Test Utilities

This package provides utilities for testing the wellness-nutrition application.

## Mock Implementations

### MockMailer

A mock implementation of the `MailerInterface` for testing email functionality without requiring a real SMTP server.

```go
import "github.com/alarmfox/wellness-nutrition/app/testutil"

func TestSomeFeature(t *testing.T) {
    mailer := testutil.NewMockMailer()

    // Use mailer in your test
    err := mailer.SendWelcomeEmail("user@example.com", "John", "https://example.com/verify")

    // Verify email was sent
    if mailer.GetEmailCount() != 1 {
        t.Error("Expected 1 email to be sent")
    }

    // Check email details
    email := mailer.GetLastEmail()
    if email.Type != "welcome" {
        t.Errorf("Expected welcome email, got %s", email.Type)
    }
}
```

**Features:**
- Records all sent emails for verification
- Can be configured to return errors
- Thread-safe for concurrent tests
- Provides helper methods to query sent emails

### MockUserRepository

A mock implementation of the UserRepository for testing without a database.

```go
func TestUserLogic(t *testing.T) {
    repo := testutil.NewMockUserRepository()

    user := &models.User{
        ID:    "user-123",
        Email: "test@example.com",
        // ... other fields
    }

    repo.Create(user)

    retrieved, err := repo.GetByID("user-123")
    if err != nil {
        t.Fatal(err)
    }
}
```

### MockBookingRepository

A mock implementation of the BookingRepository for testing without a database.

```go
func TestBookingLogic(t *testing.T) {
    repo := testutil.NewMockBookingRepository()

    booking := &models.Booking{
        UserID:       sql.NullString{String: "user-123", Valid: true},
        InstructorID: 1,
        StartsAt:     time.Now().Add(24 * time.Hour),
        Type:         models.BookingTypeSimple,
    }

    repo.Create(booking)

    bookings, err := repo.GetByUserID("user-123")
    if err != nil {
        t.Fatal(err)
    }
}
```

## Database Testing Utilities

### SetupTestDB

Creates a connection to the test database. Automatically skips the test if:
- Running with `-short` flag
- `DATABASE_URL` environment variable is not set

```go
func TestWithDatabase(t *testing.T) {
    db := testutil.SetupTestDB(t)
    defer testutil.CleanupTestDB(t, db)

    // Use db for testing
}
```

### CreateTestSchema

Creates all required database tables for testing.

```go
func TestIntegration(t *testing.T) {
    db := testutil.SetupTestDB(t)
    defer testutil.CleanupTestDB(t, db)

    testutil.CreateTestSchema(t, db)
    defer testutil.DropTestSchema(t, db)

    // Run tests with schema
}
```

### TruncateTables

Removes all data from specified tables between tests.

```go
func TestMultipleScenarios(t *testing.T) {
    db := testutil.SetupTestDB(t)
    defer testutil.CleanupTestDB(t, db)

    testutil.CreateTestSchema(t, db)
    defer testutil.DropTestSchema(t, db)

    t.Run("scenario 1", func(t *testing.T) {
        testutil.TruncateTables(t, db, "users", "bookings")
        // Test scenario 1
    })

    t.Run("scenario 2", func(t *testing.T) {
        testutil.TruncateTables(t, db, "users", "bookings")
        // Test scenario 2
    })
}
```

## Running Tests

### Unit Tests Only (No Database)

```bash
make test-unit
# or
go test -short ./...
```

### Integration Tests (Requires Database)

Set up a test database first:

```bash
# Using Docker
make test-docker-up

# Set DATABASE_URL
export DATABASE_URL="postgresql://postgres:test123@localhost:5433/test_db?sslmode=disable"

# Run integration tests
make test-integration
# or
go test -run Integration ./...

# Clean up
make test-docker-down
```

### All Tests

```bash
make test
# or
go test ./...
```

### With Coverage

```bash
make test-coverage
```

This will generate:
- `coverage.out`: Coverage data file
- `coverage.html`: HTML coverage report (open in browser)

## Best Practices

1. **Use `-short` flag for unit tests**: Integration tests should check `testing.Short()` and skip if true
2. **Clean up test data**: Use `defer` to ensure cleanup happens even if tests fail
3. **Reset mocks between tests**: Call `Reset()` on mock objects between test cases
4. **Use table-driven tests**: Structure tests with multiple scenarios using subtests
5. **Test both success and failure cases**: Include error handling tests

## Example Test Structure

```go
package mypackage_test

import (
    "testing"
    "github.com/alarmfox/wellness-nutrition/app/testutil"
)

func TestFeature(t *testing.T) {
    // Setup
    mailer := testutil.NewMockMailer()
    userRepo := testutil.NewMockUserRepository()

    // Test cases
    tests := []struct {
        name    string
        setup   func()
        verify  func(t *testing.T)
    }{
        {
            name: "success case",
            setup: func() {
                // Setup test data
            },
            verify: func(t *testing.T) {
                // Verify expectations
            },
        },
        {
            name: "error case",
            setup: func() {
                userRepo.Error = errors.New("test error")
            },
            verify: func(t *testing.T) {
                // Verify error handling
            },
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Reset mocks
            mailer.Reset()
            userRepo.Reset()

            // Run setup
            if tt.setup != nil {
                tt.setup()
            }

            // Execute test logic here

            // Verify
            if tt.verify != nil {
                tt.verify(t)
            }
        })
    }
}
```
