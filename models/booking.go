package models

import (
	"database/sql"
	"time"
)

type Booking struct {
	ID           int64
	UserID       sql.NullString
	InstructorID int64
	CreatedAt    time.Time
	StartsAt     time.Time
	Type         BookingType
}

type BookingType string

const (
	BookingTypeSimple      BookingType = "SIMPLE"
	BookingTypeMassage     BookingType = "MASSAGE"
	BookingTypeAppointment BookingType = "APPOINTMENT"
	BookingTypeDisable     BookingType = "DISABLE"
)

type BookingRepository struct {
	db *sql.DB
}

func NewBookingRepository(db *sql.DB) *BookingRepository {
	return &BookingRepository{db: db}
}

func (r *BookingRepository) GetByUserID(userID string) ([]*Booking, error) {
	query := `
		SELECT id, user_id, instructor_id, created_at, starts_at
		FROM bookings
		WHERE user_id = $1 
			AND starts_at > date_trunc('month', CURRENT_TIMESTAMP)
		ORDER BY starts_at DESC
	`

	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bookings []*Booking
	for rows.Next() {
		var booking Booking
		err := rows.Scan(
			&booking.ID,
			&booking.UserID,
			&booking.InstructorID,
			&booking.CreatedAt,
			&booking.StartsAt,
		)
		if err != nil {
			return nil, err
		}
		// Ensure times are treated as UTC since database stores TIMESTAMP (not TIMESTAMPTZ)
		booking.CreatedAt = booking.CreatedAt.UTC()
		booking.StartsAt = booking.StartsAt.UTC()
		bookings = append(bookings, &booking)
	}

	return bookings, rows.Err()
}

func (r *BookingRepository) Create(booking *Booking) error {
	query := `
		INSERT INTO bookings (user_id, instructor_id, starts_at, type)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`

	err := r.db.QueryRow(query,
		booking.UserID,
		booking.InstructorID,
		booking.StartsAt,
		booking.Type).
		Scan(&booking.ID)
	return err
}

func (r *BookingRepository) Delete(id int64) error {
	query := `DELETE from bookings WHERE id = $1`
	_, err := r.db.Exec(query, id)
	return err
}

func (r *BookingRepository) GetByID(id int64) (*Booking, error) {
	query := `
		SELECT id, user_id, instructor_id, created_at, starts_at, type
		FROM bookings
		WHERE id = $1
	`

	var booking Booking
	err := r.db.QueryRow(query, id).Scan(
		&booking.ID,
		&booking.UserID,
		&booking.InstructorID,
		&booking.CreatedAt,
		&booking.StartsAt,
		&booking.Type,
	)
	if err != nil {
		return nil, err
	}

	// Ensure times are treated as UTC since database stores TIMESTAMP (not TIMESTAMPTZ)
	booking.CreatedAt = booking.CreatedAt.UTC()
	booking.StartsAt = booking.StartsAt.UTC()

	return &booking, nil
}

func (r *BookingRepository) GetByDateRange(from, to time.Time) ([]*Booking, error) {
	query := `
		SELECT id, user_id, instructor_id, created_at, starts_at, type
		FROM bookings
		WHERE starts_at >= $1 AND starts_at <= $2
		ORDER BY starts_at ASC
	`

	rows, err := r.db.Query(query, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bookings []*Booking
	for rows.Next() {
		var booking Booking
		err := rows.Scan(
			&booking.ID,
			&booking.UserID,
			&booking.InstructorID,
			&booking.CreatedAt,
			&booking.StartsAt,
			&booking.Type,
		)
		if err != nil {
			return nil, err
		}
		// Ensure times are treated as UTC since database stores TIMESTAMP (not TIMESTAMPTZ)
		booking.CreatedAt = booking.CreatedAt.UTC()
		booking.StartsAt = booking.StartsAt.UTC()
		bookings = append(bookings, &booking)
	}

	return bookings, rows.Err()
}

func (r *BookingRepository) GetByInstructorAndDateRange(instructorID string, from, to time.Time) ([]*Booking, error) {
	query := `
		SELECT id, user_id, instructor_id, created_at, starts_at, type
		FROM bookings
		WHERE instructor_id = $1 AND starts_at >= $2 AND starts_at <= $3
		ORDER BY starts_at ASC
	`

	rows, err := r.db.Query(query, instructorID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bookings []*Booking
	for rows.Next() {
		var booking Booking
		err := rows.Scan(
			&booking.ID,
			&booking.UserID,
			&booking.InstructorID,
			&booking.CreatedAt,
			&booking.StartsAt,
			&booking.Type,
		)
		if err != nil {
			return nil, err
		}
		bookings = append(bookings, &booking)
	}

	return bookings, rows.Err()
}
