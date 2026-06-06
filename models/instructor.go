package models

import (
	"database/sql"
	"time"
)

type Instructor struct {
	ID        int64
	FirstName string
	LastName  string
	MaxSlots  int
	Enabled   bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

type InstructorRepository struct {
	db *sql.DB
}

func NewInstructorRepository(db *sql.DB) *InstructorRepository {
	return &InstructorRepository{db: db}
}

func (r *InstructorRepository) GetAll() ([]*Instructor, error) {
	query := `
		SELECT id, first_name, last_name, max_slots, enabled, created_at, updated_at
		FROM instructors
		ORDER BY first_name, last_name
	`

	return r.queryMany(query)
}

func (r *InstructorRepository) GetEnabled() ([]*Instructor, error) {
	query := `
		SELECT id, first_name, last_name, max_slots, enabled, created_at, updated_at
		FROM instructors
		WHERE enabled = TRUE
		ORDER BY first_name, last_name
	`

	return r.queryMany(query)
}

func (r *InstructorRepository) queryMany(query string, args ...interface{}) ([]*Instructor, error) {
	rows, err := r.db.Query(query, args...)
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
			&instructor.MaxSlots,
			&instructor.Enabled,
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

func (r *InstructorRepository) GetByID(id int64) (*Instructor, error) {
	query := `
		SELECT id, first_name, last_name, max_slots, enabled, created_at, updated_at
		FROM instructors
		WHERE id = $1
	`

	var instructor Instructor
	err := r.db.QueryRow(query, id).Scan(
		&instructor.ID,
		&instructor.FirstName,
		&instructor.LastName,
		&instructor.MaxSlots,
		&instructor.Enabled,
		&instructor.CreatedAt,
		&instructor.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &instructor, nil
}

func (r *InstructorRepository) GetEnabledByID(id int64) (*Instructor, error) {
	query := `
		SELECT id, first_name, last_name, max_slots, enabled, created_at, updated_at
		FROM instructors
		WHERE id = $1 AND enabled = TRUE
	`

	var instructor Instructor
	err := r.db.QueryRow(query, id).Scan(
		&instructor.ID,
		&instructor.FirstName,
		&instructor.LastName,
		&instructor.MaxSlots,
		&instructor.Enabled,
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
		INSERT INTO instructors (first_name, last_name, max_slots, enabled)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at
	`

	err := r.db.QueryRow(query,
		instructor.FirstName,
		instructor.LastName,
		instructor.MaxSlots,
		instructor.Enabled,
	).Scan(&instructor.ID, &instructor.CreatedAt, &instructor.UpdatedAt)

	return err
}

func (r *InstructorRepository) Update(instructor *Instructor) error {
	query := `
		UPDATE instructors
		SET first_name = $2, last_name = $3, max_slots = $4, enabled = $5, updated_at = $6
		WHERE id = $1
	`

	_, err := r.db.Exec(query,
		instructor.ID,
		instructor.FirstName,
		instructor.LastName,
		instructor.MaxSlots,
		instructor.Enabled,
		time.Now().UTC(),
	)

	return err
}

func (r *InstructorRepository) Delete(id int64) error {
	query := `DELETE FROM instructors WHERE id = $1`
	_, err := r.db.Exec(query, id)
	return err
}
