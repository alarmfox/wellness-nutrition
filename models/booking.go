package models

import (
	"database/sql"
	"time"
)

type Booking struct {
	ID        int64
	UserID    string
	CreatedAt time.Time
	StartsAt  time.Time
}

type Slot struct {
	StartsAt    time.Time
	PeopleCount int
	Disabled    bool
}

type EventType string

const (
	EventTypeCreated        EventType = "CREATED"
	EventTypeDeleted        EventType = "DELETED"
	EventTypeBookingCreated EventType = "BOOKING_CREATED"
	EventTypeSlotDisabled   EventType = "SLOT_DISABLED"
	EventTypeSlotEnabled    EventType = "SLOT_ENABLED"
)

type Event struct {
	ID         int
	UserID     string
	StartsAt   time.Time
	Type       EventType
	OccurredAt time.Time
}

type BookingRepository struct {
	db *sql.DB
}

func NewBookingRepository(db *sql.DB) *BookingRepository {
	return &BookingRepository{db: db}
}

func (r *BookingRepository) GetByUserID(userID string) ([]*Booking, error) {
	query := `
		SELECT id, user_id, created_at, starts_at
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

func (r *BookingRepository) GetByDateRange(from, to time.Time) ([]*Booking, error) {
	query := `
		SELECT id, user_id, created_at, starts_at
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
		INSERT INTO bookings (user_id, created_at, starts_at)
		VALUES ($1, $2, $3)
		RETURNING id
	`
	
	err := r.db.QueryRow(query, booking.UserID, booking.CreatedAt, booking.StartsAt).Scan(&booking.ID)
	return err
}

func (r *BookingRepository) Delete(id int64) error {
	query := `DELETE FROM bookings WHERE id = $1`
	_, err := r.db.Exec(query, id)
	return err
}

func (r *BookingRepository) GetBySlotTime(startsAt time.Time) ([]*Booking, error) {
	query := `
		SELECT id, user_id, created_at, starts_at
		FROM bookings
		WHERE starts_at = $1
		ORDER BY created_at ASC
	`
	
	rows, err := r.db.Query(query, startsAt)
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

func (r *BookingRepository) GetByID(id int64) (*Booking, error) {
	query := `
		SELECT id, user_id, created_at, starts_at
		FROM bookings
		WHERE id = $1
	`
	
	var booking Booking
	err := r.db.QueryRow(query, id).Scan(
		&booking.ID,
		&booking.UserID,
		&booking.CreatedAt,
		&booking.StartsAt,
	)
	
	if err != nil {
		return nil, err
	}
	
	return &booking, nil
}

type SlotRepository struct {
	db *sql.DB
}

func NewSlotRepository(db *sql.DB) *SlotRepository {
	return &SlotRepository{db: db}
}

func (r *SlotRepository) GetAvailableSlots(from, to time.Time) ([]*Slot, error) {
	query := `
		SELECT starts_at, people_count, disabled
		FROM slots
		WHERE starts_at >= $1 AND starts_at < $2
			AND disabled = false
		ORDER BY starts_at ASC
	`
	
	rows, err := r.db.Query(query, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var slots []*Slot
	for rows.Next() {
		var slot Slot
		err := rows.Scan(
			&slot.StartsAt,
			&slot.PeopleCount,
			&slot.Disabled,
		)
		if err != nil {
			return nil, err
		}
		slots = append(slots, &slot)
	}
	
	return slots, rows.Err()
}

// GetSlotsByDateRange returns all slots (including disabled) in a date range
func (r *SlotRepository) GetSlotsByDateRange(from, to time.Time) ([]*Slot, error) {
	query := `
		SELECT starts_at, people_count, disabled
		FROM slots
		WHERE starts_at >= $1 AND starts_at < $2
		ORDER BY starts_at ASC
	`
	
	rows, err := r.db.Query(query, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var slots []*Slot
	for rows.Next() {
		var slot Slot
		err := rows.Scan(
			&slot.StartsAt,
			&slot.PeopleCount,
			&slot.Disabled,
		)
		if err != nil {
			return nil, err
		}
		slots = append(slots, &slot)
	}
	
	return slots, rows.Err()
}

func (r *SlotRepository) GetByTime(startsAt time.Time) (*Slot, error) {
	query := `
		SELECT starts_at, people_count, disabled
		FROM slots
		WHERE starts_at = $1
	`
	
	var slot Slot
	err := r.db.QueryRow(query, startsAt).Scan(
		&slot.StartsAt,
		&slot.PeopleCount,
		&slot.Disabled,
	)
	
	if err != nil {
		return nil, err
	}
	
	return &slot, nil
}

func (r *SlotRepository) IncrementPeopleCount(startsAt time.Time) error {
	query := `
		UPDATE slots
		SET people_count = people_count + 1
		WHERE starts_at = $1
	`
	_, err := r.db.Exec(query, startsAt)
	return err
}

func (r *SlotRepository) DecrementPeopleCount(startsAt time.Time) error {
	query := `
		UPDATE slots
		SET people_count = people_count - 1
		WHERE starts_at = $1 AND people_count > 0
	`
	_, err := r.db.Exec(query, startsAt)
	return err
}

func (r *SlotRepository) Update(slot *Slot) error {
	query := `
		UPDATE slots
		SET disabled = $1, people_count = $2
		WHERE starts_at = $3
	`
	_, err := r.db.Exec(query, slot.Disabled, slot.PeopleCount, slot.StartsAt)
	return err
}

func (r *SlotRepository) Create(slot *Slot) error {
	query := `
		INSERT INTO slots (starts_at, people_count, disabled)
		VALUES ($1, $2, $3)
	`
	_, err := r.db.Exec(query, slot.StartsAt, slot.PeopleCount, slot.Disabled)
	return err
}

type EventRepository struct {
	db *sql.DB
}

func NewEventRepository(db *sql.DB) *EventRepository {
	return &EventRepository{db: db}
}

func (r *EventRepository) Create(event *Event) error {
	query := `
		INSERT INTO events (user_id, starts_at, type, occurred_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`
	
	err := r.db.QueryRow(query, event.UserID, event.StartsAt, event.Type, event.OccurredAt).Scan(&event.ID)
	return err
}

func (r *EventRepository) GetAll() ([]*Event, error) {
	query := `
		SELECT id, user_id, starts_at, type, occurred_at
		FROM events
		ORDER BY occurred_at DESC
		LIMIT 100
	`
	
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var events []*Event
	for rows.Next() {
		var event Event
		err := rows.Scan(
			&event.ID,
			&event.UserID,
			&event.StartsAt,
			&event.Type,
			&event.OccurredAt,
		)
		if err != nil {
			return nil, err
		}
		events = append(events, &event)
	}
	
	return events, rows.Err()
}
