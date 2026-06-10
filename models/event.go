package models

import (
	"database/sql"
	"time"
)

type EventType string

const (
	EventTypeCreated         EventType = "CREATED"
	EventTypeDeleted         EventType = "DELETED"
	EventTypeBookingCreated  EventType = "BOOKING_CREATED"
	EventTypeSlotDisabled    EventType = "SLOT_DISABLED"
	EventTypeSlotEnabled     EventType = "SLOT_ENABLED"
	EventTypeSlotMassage     EventType = "SLOT_MASSAGE"
	EventTypeSlotAppointment EventType = "SLOT_APPOINTMENT"
	EventTypeSlotUnreserved  EventType = "SLOT_UNRESERVED"
)

type Event struct {
	ID         int
	UserID     string
	StartsAt   time.Time
	Type       EventType
	OccurredAt time.Time
}

type EventWithUser struct {
	ID            int
	UserID        string
	StartsAt      time.Time
	Type          EventType
	OccurredAt    time.Time
	UserFirstName sql.NullString
	UserLastName  sql.NullString
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

func (r *EventRepository) GetAllWithUsers() ([]*EventWithUser, error) {
	query := `
		SELECT e.id, e.user_id, e.starts_at, e.type, e.occurred_at,
			   u.first_name, u.last_name
		FROM events e
		LEFT JOIN users u ON u.id = e.user_id
		ORDER BY e.occurred_at DESC
		LIMIT 100
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*EventWithUser
	for rows.Next() {
		var event EventWithUser
		err := rows.Scan(
			&event.ID,
			&event.UserID,
			&event.StartsAt,
			&event.Type,
			&event.OccurredAt,
			&event.UserFirstName,
			&event.UserLastName,
		)
		if err != nil {
			return nil, err
		}
		events = append(events, &event)
	}

	return events, rows.Err()
}
