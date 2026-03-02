package testutil

import (
	"sync"
	"time"

	"github.com/alarmfox/wellness-nutrition/app/mail"
)

// MockMailer is a mock implementation of the MailerInterface for testing
type MockMailer struct {
	mu     sync.Mutex
	Emails []SentEmail
	Error  error // If set, all methods will return this error
}

// SentEmail represents an email that was sent through the mock mailer
type SentEmail struct {
	To      string
	Subject string
	Data    mail.EmailData
	Type    string // "generic", "welcome", "reset", "new_booking", "delete_booking", "reminder"
}

// NewMockMailer creates a new mock mailer
func NewMockMailer() *MockMailer {
	return &MockMailer{
		Emails: make([]SentEmail, 0),
	}
}

// SendEmail records a generic email
func (m *MockMailer) SendEmail(to, subject string, data mail.EmailData) error {
	if m.Error != nil {
		return m.Error
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.Emails = append(m.Emails, SentEmail{
		To:      to,
		Subject: subject,
		Data:    data,
		Type:    "generic",
	})

	return nil
}

// SendWelcomeEmail records a welcome email
func (m *MockMailer) SendWelcomeEmail(email, firstName, verificationURL string) error {
	if m.Error != nil {
		return m.Error
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.Emails = append(m.Emails, SentEmail{
		To:      email,
		Subject: "Benvenuto in Wellness & Nutrition",
		Data: mail.EmailData{
			Name:       firstName,
			ButtonLink: verificationURL,
		},
		Type: "welcome",
	})

	return nil
}

// SendResetEmail records a password reset email
func (m *MockMailer) SendResetEmail(email, firstName, verificationURL string) error {
	if m.Error != nil {
		return m.Error
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.Emails = append(m.Emails, SentEmail{
		To:      email,
		Subject: "Ripristino password",
		Data: mail.EmailData{
			Name:       firstName,
			ButtonLink: verificationURL,
		},
		Type: "reset",
	})

	return nil
}

// SendNewBookingNotification records a new booking notification
func (m *MockMailer) SendNewBookingNotification(firstName, lastName string, startsAt time.Time) error {
	if m.Error != nil {
		return m.Error
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.Emails = append(m.Emails, SentEmail{
		To:      "admin@example.com", // In real implementation, this comes from env var
		Subject: "Nuova prenotazione",
		Data: mail.EmailData{
			Name: "amministratore",
		},
		Type: "new_booking",
	})

	return nil
}

// SendDeleteBookingNotification records a booking deletion notification
func (m *MockMailer) SendDeleteBookingNotification(firstName, lastName string, startsAt time.Time) error {
	if m.Error != nil {
		return m.Error
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.Emails = append(m.Emails, SentEmail{
		To:      "admin@example.com", // In real implementation, this comes from env var
		Subject: "Prenotazione cancellata",
		Data: mail.EmailData{
			Name: "amministratore",
		},
		Type: "delete_booking",
	})

	return nil
}

// SendReminderEmail records a reminder email
func (m *MockMailer) SendReminderEmail(email, firstName string, startsAt time.Time) error {
	if m.Error != nil {
		return m.Error
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.Emails = append(m.Emails, SentEmail{
		To:      email,
		Subject: "Promemoria prenotazione - Wellness & Nutrition",
		Data: mail.EmailData{
			Name: firstName,
		},
		Type: "reminder",
	})

	return nil
}

// Reset clears all recorded emails
func (m *MockMailer) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Emails = make([]SentEmail, 0)
	m.Error = nil
}

// GetEmails returns a copy of all sent emails
func (m *MockMailer) GetEmails() []SentEmail {
	m.mu.Lock()
	defer m.mu.Unlock()

	emails := make([]SentEmail, len(m.Emails))
	copy(emails, m.Emails)
	return emails
}

// GetEmailCount returns the number of emails sent
func (m *MockMailer) GetEmailCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.Emails)
}

// GetLastEmail returns the last email sent, or nil if none
func (m *MockMailer) GetLastEmail() *SentEmail {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.Emails) == 0 {
		return nil
	}

	email := m.Emails[len(m.Emails)-1]
	return &email
}

// GetEmailsByType returns all emails of a specific type
func (m *MockMailer) GetEmailsByType(emailType string) []SentEmail {
	m.mu.Lock()
	defer m.mu.Unlock()

	var filtered []SentEmail
	for _, email := range m.Emails {
		if email.Type == emailType {
			filtered = append(filtered, email)
		}
	}
	return filtered
}
