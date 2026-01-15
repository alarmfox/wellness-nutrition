package mail

import (
	"time"
)

// MailerInterface defines the interface for sending emails
// This allows for easy mocking in tests
type MailerInterface interface {
	SendEmail(to, subject string, data EmailData) error
	SendWelcomeEmail(email, firstName, verificationURL string) error
	SendResetEmail(email, firstName, verificationURL string) error
	SendNewBookingNotification(firstName, lastName string, startsAt time.Time) error
	SendDeleteBookingNotification(firstName, lastName string, startsAt time.Time) error
	SendReminderEmail(email, firstName string, startsAt time.Time) error
}

// Ensure Mailer implements MailerInterface
var _ MailerInterface = (*Mailer)(nil)
