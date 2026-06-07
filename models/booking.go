package models

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"hash/fnv"
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

var (
	ErrSlotUnavailable = errors.New("slot unavailable")
	ErrNoAccesses      = errors.New("no remaining accesses")
)

type BookingWithUser struct {
	ID            int64
	UserID        sql.NullString
	InstructorID  int64
	CreatedAt     time.Time
	StartsAt      time.Time
	Type          BookingType
	UserFirstName sql.NullString
	UserLastName  sql.NullString
	UserEmail     sql.NullString
	UserSubType   sql.NullString
}

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

func (r *BookingRepository) CreateUserBooking(booking *Booking, neededSlots, maxSlots int) error {
	tx, err := r.db.BeginTx(context.Background(), &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`SELECT pg_advisory_xact_lock($1)`, bookingLockKey(booking.InstructorID, booking.StartsAt)); err != nil {
		return err
	}

	rows, err := tx.Query(`
		SELECT b.type, COALESCE(u.sub_type, '')
		FROM bookings b
		LEFT JOIN users u ON u.id = b.user_id
		WHERE b.instructor_id = $1 AND b.starts_at = $2
		FOR UPDATE OF b
	`, booking.InstructorID, booking.StartsAt)
	if err != nil {
		return err
	}

	usedSlots := 0
	for rows.Next() {
		var bookingType BookingType
		var subType string
		if err := rows.Scan(&bookingType, &subType); err != nil {
			rows.Close()
			return err
		}

		if bookingType == BookingTypeDisable || bookingType == BookingTypeMassage || bookingType == BookingTypeAppointment {
			rows.Close()
			return ErrSlotUnavailable
		}

		if bookingType == BookingTypeSimple {
			if subType == string(SubTypeSingle) {
				usedSlots += 2
			} else {
				usedSlots++
			}
		}
	}
	if err := rows.Close(); err != nil {
		return err
	}
	if err := rows.Err(); err != nil {
		return err
	}

	if usedSlots+neededSlots > maxSlots {
		return ErrSlotUnavailable
	}

	result, err := tx.Exec(
		`UPDATE users SET remaining_accesses = remaining_accesses - 1 WHERE id = $1 AND remaining_accesses > 0`,
		booking.UserID.String,
	)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrNoAccesses
	}

	err = tx.QueryRow(`
		INSERT INTO bookings (user_id, instructor_id, starts_at, type)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, booking.UserID, booking.InstructorID, booking.StartsAt, booking.Type).Scan(&booking.ID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *BookingRepository) Delete(id int64) error {
	query := `DELETE from bookings WHERE id = $1`
	_, err := r.db.Exec(query, id)
	return err
}

func (r *BookingRepository) GetWithUsersByDateRange(from, to time.Time) ([]*BookingWithUser, error) {
	query := `
		SELECT b.id, b.user_id, b.instructor_id, b.created_at, b.starts_at, b.type,
			   u.first_name, u.last_name, u.email, u.sub_type
		FROM bookings b
		LEFT JOIN users u ON u.id = b.user_id
		WHERE b.starts_at >= $1 AND b.starts_at <= $2
		ORDER BY b.starts_at ASC
	`

	return r.queryWithUsers(query, from, to)
}

func (r *BookingRepository) GetWithUsersByInstructorAndDateRange(instructorID string, from, to time.Time) ([]*BookingWithUser, error) {
	query := `
		SELECT b.id, b.user_id, b.instructor_id, b.created_at, b.starts_at, b.type,
			   u.first_name, u.last_name, u.email, u.sub_type
		FROM bookings b
		LEFT JOIN users u ON u.id = b.user_id
		WHERE b.instructor_id = $1 AND b.starts_at >= $2 AND b.starts_at <= $3
		ORDER BY b.starts_at ASC
	`

	return r.queryWithUsers(query, instructorID, from, to)
}

func (r *BookingRepository) queryWithUsers(query string, args ...interface{}) ([]*BookingWithUser, error) {
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bookings []*BookingWithUser
	for rows.Next() {
		var booking BookingWithUser
		err := rows.Scan(
			&booking.ID,
			&booking.UserID,
			&booking.InstructorID,
			&booking.CreatedAt,
			&booking.StartsAt,
			&booking.Type,
			&booking.UserFirstName,
			&booking.UserLastName,
			&booking.UserEmail,
			&booking.UserSubType,
		)
		if err != nil {
			return nil, err
		}
		bookings = append(bookings, &booking)
	}

	return bookings, rows.Err()
}

func bookingLockKey(instructorID int64, startsAt time.Time) int64 {
	h := fnv.New64a()
	fmt.Fprintf(h, "%d:%d", instructorID, startsAt.Unix())
	return int64(h.Sum64())
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
