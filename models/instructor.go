package models

import (
	"database/sql"
	"time"
)

type Instructor struct {
	ID        string
	FirstName string
	LastName  string
	Email     string
	Password  sql.NullString
	CreatedAt time.Time
	UpdatedAt time.Time
}

type InstructorSlot struct {
	InstructorID string
	StartsAt     time.Time
	PeopleCount  int
	MaxCapacity  int
}

type InstructorRepository struct {
	db *sql.DB
}

func NewInstructorRepository(db *sql.DB) *InstructorRepository {
	return &InstructorRepository{db: db}
}

func (r *InstructorRepository) GetAll() ([]*Instructor, error) {
	query := `
		SELECT id, first_name, last_name, email, password, created_at, updated_at
		FROM instructors
		ORDER BY first_name, last_name
	`
	
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var instructors []*Instructor
	for rows.Next() {
		var instructor Instructor
		err := rows.Scan(
			&instructor.ID,
			&instructor.FirstName,
			&instructor.LastName,
			&instructor.Email,
			&instructor.Password,
			&instructor.CreatedAt,
			&instructor.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		instructors = append(instructors, &instructor)
	}
	
	return instructors, rows.Err()
}

func (r *InstructorRepository) GetByID(id string) (*Instructor, error) {
	query := `
		SELECT id, first_name, last_name, email, password, created_at, updated_at
		FROM instructors
		WHERE id = $1
	`
	
	var instructor Instructor
	err := r.db.QueryRow(query, id).Scan(
		&instructor.ID,
		&instructor.FirstName,
		&instructor.LastName,
		&instructor.Email,
		&instructor.Password,
		&instructor.CreatedAt,
		&instructor.UpdatedAt,
	)
	
	if err != nil {
		return nil, err
	}
	
	return &instructor, nil
}

func (r *InstructorRepository) GetByEmail(email string) (*Instructor, error) {
	query := `
		SELECT id, first_name, last_name, email, password, created_at, updated_at
		FROM instructors
		WHERE LOWER(email) = LOWER($1)
	`
	
	var instructor Instructor
	err := r.db.QueryRow(query, email).Scan(
		&instructor.ID,
		&instructor.FirstName,
		&instructor.LastName,
		&instructor.Email,
		&instructor.Password,
		&instructor.CreatedAt,
		&instructor.UpdatedAt,
	)
	
	if err != nil {
		return nil, err
	}
	
	return &instructor, nil
}

func (r *InstructorRepository) Create(instructor *Instructor) error {
	query := `
		INSERT INTO instructors (id, first_name, last_name, email, password, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	
	_, err := r.db.Exec(query,
		instructor.ID,
		instructor.FirstName,
		instructor.LastName,
		instructor.Email,
		instructor.Password,
		instructor.CreatedAt,
		instructor.UpdatedAt,
	)
	
	return err
}

func (r *InstructorRepository) Update(instructor *Instructor) error {
	query := `
		UPDATE instructors
		SET first_name = $2, last_name = $3, email = $4, password = $5, updated_at = $6
		WHERE id = $1
	`
	
	_, err := r.db.Exec(query,
		instructor.ID,
		instructor.FirstName,
		instructor.LastName,
		instructor.Email,
		instructor.Password,
		time.Now(),
	)
	
	return err
}

func (r *InstructorRepository) Delete(id string) error {
	query := `DELETE FROM instructors WHERE id = $1`
	_, err := r.db.Exec(query, id)
	return err
}

// InstructorSlot management
type InstructorSlotRepository struct {
	db *sql.DB
}

func NewInstructorSlotRepository(db *sql.DB) *InstructorSlotRepository {
	return &InstructorSlotRepository{db: db}
}

func (r *InstructorSlotRepository) GetByInstructorAndTime(instructorID string, startsAt time.Time) (*InstructorSlot, error) {
	query := `
		SELECT instructor_id, starts_at, people_count, max_capacity
		FROM instructor_slots
		WHERE instructor_id = $1 AND starts_at = $2
	`
	
	var slot InstructorSlot
	err := r.db.QueryRow(query, instructorID, startsAt).Scan(
		&slot.InstructorID,
		&slot.StartsAt,
		&slot.PeopleCount,
		&slot.MaxCapacity,
	)
	
	if err != nil {
		return nil, err
	}
	
	return &slot, nil
}

func (r *InstructorSlotRepository) Create(slot *InstructorSlot) error {
	query := `
		INSERT INTO instructor_slots (instructor_id, starts_at, people_count, max_capacity)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (instructor_id, starts_at) DO NOTHING
	`
	
	_, err := r.db.Exec(query,
		slot.InstructorID,
		slot.StartsAt,
		slot.PeopleCount,
		slot.MaxCapacity,
	)
	
	return err
}

func (r *InstructorSlotRepository) IncrementPeopleCount(instructorID string, startsAt time.Time) error {
	query := `
		UPDATE instructor_slots
		SET people_count = people_count + 1
		WHERE instructor_id = $1 AND starts_at = $2
	`
	_, err := r.db.Exec(query, instructorID, startsAt)
	return err
}

func (r *InstructorSlotRepository) DecrementPeopleCount(instructorID string, startsAt time.Time) error {
	query := `
		UPDATE instructor_slots
		SET people_count = people_count - 1
		WHERE instructor_id = $1 AND starts_at = $2 AND people_count > 0
	`
	_, err := r.db.Exec(query, instructorID, startsAt)
	return err
}

func (r *InstructorSlotRepository) GetAvailableForInstructor(instructorID string, from, to time.Time) ([]*InstructorSlot, error) {
	query := `
		SELECT instructor_id, starts_at, people_count, max_capacity
		FROM instructor_slots
		WHERE instructor_id = $1 
			AND starts_at >= $2 
			AND starts_at < $3
			AND people_count < max_capacity
		ORDER BY starts_at ASC
	`
	
	rows, err := r.db.Query(query, instructorID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var slots []*InstructorSlot
	for rows.Next() {
		var slot InstructorSlot
		err := rows.Scan(
			&slot.InstructorID,
			&slot.StartsAt,
			&slot.PeopleCount,
			&slot.MaxCapacity,
		)
		if err != nil {
			return nil, err
		}
		slots = append(slots, &slot)
	}
	
	return slots, rows.Err()
}
