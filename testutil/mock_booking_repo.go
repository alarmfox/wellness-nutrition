package testutil

import (
	"database/sql"
	"sync"
	"time"

	"github.com/alarmfox/wellness-nutrition/app/models"
)

// MockBookingRepository is a mock implementation of BookingRepository for testing
type MockBookingRepository struct {
	mu       sync.RWMutex
	bookings map[int64]*models.Booking
	nextID   int64
	Error    error
}

// NewMockBookingRepository creates a new mock booking repository
func NewMockBookingRepository() *MockBookingRepository {
	return &MockBookingRepository{
		bookings: make(map[int64]*models.Booking),
		nextID:   1,
	}
}

func (m *MockBookingRepository) GetByUserID(userID string) ([]*models.Booking, error) {
	if m.Error != nil {
		return nil, m.Error
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	var bookings []*models.Booking
	now := time.Now()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	for _, booking := range m.bookings {
		if booking.UserID.Valid && booking.UserID.String == userID && booking.StartsAt.After(startOfMonth) {
			bookings = append(bookings, booking)
		}
	}
	return bookings, nil
}

func (m *MockBookingRepository) Create(booking *models.Booking) error {
	if m.Error != nil {
		return m.Error
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	booking.ID = m.nextID
	m.nextID++
	booking.CreatedAt = time.Now().UTC()
	m.bookings[booking.ID] = booking
	return nil
}

func (m *MockBookingRepository) Delete(id int64) error {
	if m.Error != nil {
		return m.Error
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.bookings[id]; !ok {
		return sql.ErrNoRows
	}

	delete(m.bookings, id)
	return nil
}

func (m *MockBookingRepository) GetByID(id int64) (*models.Booking, error) {
	if m.Error != nil {
		return nil, m.Error
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	booking, ok := m.bookings[id]
	if !ok {
		return nil, sql.ErrNoRows
	}
	return booking, nil
}

func (m *MockBookingRepository) GetByDateRange(from, to time.Time) ([]*models.Booking, error) {
	if m.Error != nil {
		return nil, m.Error
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	var bookings []*models.Booking
	for _, booking := range m.bookings {
		if (booking.StartsAt.Equal(from) || booking.StartsAt.After(from)) &&
			(booking.StartsAt.Equal(to) || booking.StartsAt.Before(to)) {
			bookings = append(bookings, booking)
		}
	}
	return bookings, nil
}

func (m *MockBookingRepository) GetByInstructorAndDateRange(instructorID string, from, to time.Time) ([]*models.Booking, error) {
	if m.Error != nil {
		return nil, m.Error
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	var bookings []*models.Booking
	for _, booking := range m.bookings {
		// Convert instructorID to int64 for comparison
		var instrID int64
		// Simple conversion - in real code this would be more robust
		if instructorID == "1" {
			instrID = 1
		}

		if booking.InstructorID == instrID &&
			(booking.StartsAt.Equal(from) || booking.StartsAt.After(from)) &&
			(booking.StartsAt.Equal(to) || booking.StartsAt.Before(to)) {
			bookings = append(bookings, booking)
		}
	}
	return bookings, nil
}

// Reset clears all bookings and errors
func (m *MockBookingRepository) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.bookings = make(map[int64]*models.Booking)
	m.nextID = 1
	m.Error = nil
}

// AddBooking adds a booking to the mock repository (for test setup)
func (m *MockBookingRepository) AddBooking(booking *models.Booking) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if booking.ID == 0 {
		booking.ID = m.nextID
		m.nextID++
	}
	m.bookings[booking.ID] = booking
}

// GetBookingCount returns the number of bookings in the mock repository
func (m *MockBookingRepository) GetBookingCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.bookings)
}
