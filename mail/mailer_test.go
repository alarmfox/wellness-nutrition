package mail_test

import (
	"testing"
	"time"

	"github.com/alarmfox/wellness-nutrition/app/testutil"
)

func TestMockMailer(t *testing.T) {
	mailer := testutil.NewMockMailer()

	t.Run("Send Welcome Email", func(t *testing.T) {
		mailer.Reset()

		err := mailer.SendWelcomeEmail("test@example.com", "John", "https://example.com/verify")
		if err != nil {
			t.Fatalf("Failed to send welcome email: %v", err)
		}

		if mailer.GetEmailCount() != 1 {
			t.Errorf("Expected 1 email, got %d", mailer.GetEmailCount())
		}

		email := mailer.GetLastEmail()
		if email == nil {
			t.Fatal("Expected email to be sent")
		}

		if email.To != "test@example.com" {
			t.Errorf("Expected recipient test@example.com, got %s", email.To)
		}

		if email.Type != "welcome" {
			t.Errorf("Expected type welcome, got %s", email.Type)
		}

		if email.Data.Name != "John" {
			t.Errorf("Expected name John, got %s", email.Data.Name)
		}
	})

	t.Run("Send Reset Email", func(t *testing.T) {
		mailer.Reset()

		err := mailer.SendResetEmail("user@example.com", "Jane", "https://example.com/reset")
		if err != nil {
			t.Fatalf("Failed to send reset email: %v", err)
		}

		email := mailer.GetLastEmail()
		if email == nil {
			t.Fatal("Expected email to be sent")
		}

		if email.Type != "reset" {
			t.Errorf("Expected type reset, got %s", email.Type)
		}

		if email.Subject != "Ripristino password" {
			t.Errorf("Expected subject 'Ripristino password', got %s", email.Subject)
		}
	})

	t.Run("Send New Booking Notification", func(t *testing.T) {
		mailer.Reset()

		startsAt := time.Now().Add(24 * time.Hour)
		err := mailer.SendNewBookingNotification("John", "Doe", startsAt)
		if err != nil {
			t.Fatalf("Failed to send booking notification: %v", err)
		}

		email := mailer.GetLastEmail()
		if email == nil {
			t.Fatal("Expected email to be sent")
		}

		if email.Type != "new_booking" {
			t.Errorf("Expected type new_booking, got %s", email.Type)
		}
	})

	t.Run("Send Delete Booking Notification", func(t *testing.T) {
		mailer.Reset()

		startsAt := time.Now().Add(24 * time.Hour)
		err := mailer.SendDeleteBookingNotification("John", "Doe", startsAt)
		if err != nil {
			t.Fatalf("Failed to send delete notification: %v", err)
		}

		email := mailer.GetLastEmail()
		if email == nil {
			t.Fatal("Expected email to be sent")
		}

		if email.Type != "delete_booking" {
			t.Errorf("Expected type delete_booking, got %s", email.Type)
		}
	})

	t.Run("Send Reminder Email", func(t *testing.T) {
		mailer.Reset()

		startsAt := time.Now().Add(24 * time.Hour)
		err := mailer.SendReminderEmail("user@example.com", "John", startsAt)
		if err != nil {
			t.Fatalf("Failed to send reminder email: %v", err)
		}

		email := mailer.GetLastEmail()
		if email == nil {
			t.Fatal("Expected email to be sent")
		}

		if email.Type != "reminder" {
			t.Errorf("Expected type reminder, got %s", email.Type)
		}
	})

	t.Run("Multiple Emails", func(t *testing.T) {
		mailer.Reset()

		// Send multiple emails
		mailer.SendWelcomeEmail("user1@example.com", "User1", "url1")
		mailer.SendWelcomeEmail("user2@example.com", "User2", "url2")
		mailer.SendResetEmail("user3@example.com", "User3", "url3")

		if mailer.GetEmailCount() != 3 {
			t.Errorf("Expected 3 emails, got %d", mailer.GetEmailCount())
		}

		// Get emails by type
		welcomeEmails := mailer.GetEmailsByType("welcome")
		if len(welcomeEmails) != 2 {
			t.Errorf("Expected 2 welcome emails, got %d", len(welcomeEmails))
		}

		resetEmails := mailer.GetEmailsByType("reset")
		if len(resetEmails) != 1 {
			t.Errorf("Expected 1 reset email, got %d", len(resetEmails))
		}
	})

	t.Run("Error Handling", func(t *testing.T) {
		mailer.Reset()

		// Set an error
		testErr := &mockError{msg: "test error"}
		mailer.Error = testErr

		// All send methods should return the error
		err := mailer.SendWelcomeEmail("test@example.com", "Test", "url")
		if err != testErr {
			t.Errorf("Expected test error, got %v", err)
		}

		// No email should be recorded
		if mailer.GetEmailCount() != 0 {
			t.Errorf("Expected 0 emails when error is set, got %d", mailer.GetEmailCount())
		}
	})
}

// mockError is a simple error type for testing
type mockError struct {
	msg string
}

func (e *mockError) Error() string {
	return e.msg
}
