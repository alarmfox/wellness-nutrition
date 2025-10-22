package models

import (
	"database/sql"
	"fmt"
	"time"
)

type SubType string

const (
	SubTypeShared SubType = "SHARED"
	SubTypeSingle SubType = "SINGLE"
)

type User struct {
	ID                         string
	FirstName                  string
	LastName                   sql.NullString
	Address                    string
	Password                   sql.NullString
	MedOk                      bool
	Cellphone                  sql.NullString
	SubType                    SubType
	Email                      string
	EmailVerified              sql.NullTime
	ExpiresAt                  time.Time
	RemainingAccesses          int
	VerificationToken          sql.NullString
	VerificationTokenExpiresIn sql.NullTime
	Goals                      sql.NullString
}

type Admin struct {
	ID        string
	FirstName string
	LastName  sql.NullString
	Email     string
	Password  string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) GetByEmail(email string) (*User, error) {
	query := `
		SELECT id, first_name, last_name, address, password, med_ok, 
			   cellphone, sub_type, email, email_verified, expires_at, 
			   remaining_accesses, verification_token, verification_token_expires_in, goals
		FROM users
		WHERE LOWER(email) = LOWER($1)
	`

	var user User
	err := r.db.QueryRow(query, email).Scan(
		&user.ID,
		&user.FirstName,
		&user.LastName,
		&user.Address,
		&user.Password,
		&user.MedOk,
		&user.Cellphone,
		&user.SubType,
		&user.Email,
		&user.EmailVerified,
		&user.ExpiresAt,
		&user.RemainingAccesses,
		&user.VerificationToken,
		&user.VerificationTokenExpiresIn,
		&user.Goals,
	)

	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *UserRepository) GetByID(id string) (*User, error) {
	query := `
		SELECT id, first_name, last_name, address, password, med_ok, 
			   cellphone, sub_type, email, email_verified, expires_at, 
			   remaining_accesses, verification_token, verification_token_expires_in, goals
		FROM users
		WHERE id = $1
	`

	var user User
	err := r.db.QueryRow(query, id).Scan(
		&user.ID,
		&user.FirstName,
		&user.LastName,
		&user.Address,
		&user.Password,
		&user.MedOk,
		&user.Cellphone,
		&user.SubType,
		&user.Email,
		&user.EmailVerified,
		&user.ExpiresAt,
		&user.RemainingAccesses,
		&user.VerificationToken,
		&user.VerificationTokenExpiresIn,
		&user.Goals,
	)

	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *UserRepository) GetByVerificationToken(token string) (*User, error) {
	query := `
		SELECT id, first_name, last_name, address, password, med_ok, 
			   cellphone, sub_type, email, email_verified, expires_at, 
			   remaining_accesses, verification_token, verification_token_expires_in, goals
		FROM users
		WHERE verification_token = $1
	`

	var user User
	err := r.db.QueryRow(query, token).Scan(
		&user.ID,
		&user.FirstName,
		&user.LastName,
		&user.Address,
		&user.Password,
		&user.MedOk,
		&user.Cellphone,
		&user.SubType,
		&user.Email,
		&user.EmailVerified,
		&user.ExpiresAt,
		&user.RemainingAccesses,
		&user.VerificationToken,
		&user.VerificationTokenExpiresIn,
		&user.Goals,
	)

	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *UserRepository) GetAll() ([]*User, error) {
	query := `
		SELECT id, first_name, last_name, address, password, med_ok, 
			   cellphone, sub_type, email, email_verified, expires_at, 
			   remaining_accesses, verification_token, verification_token_expires_in, goals
		FROM users
		ORDER BY first_name, last_name
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		var user User
		err := rows.Scan(
			&user.ID,
			&user.FirstName,
			&user.LastName,
			&user.Address,
			&user.Password,
			&user.MedOk,
			&user.Cellphone,
			&user.SubType,
			&user.Email,
			&user.EmailVerified,
			&user.ExpiresAt,
			&user.RemainingAccesses,
			&user.VerificationToken,
			&user.VerificationTokenExpiresIn,
			&user.Goals,
		)
		if err != nil {
			return nil, err
		}
		users = append(users, &user)
	}

	return users, rows.Err()
}

func (r *UserRepository) Create(user *User) error {
	query := `
		INSERT INTO users 
			(id, first_name, last_name, address, password, med_ok, 
			 cellphone, sub_type, email, email_verified, expires_at, 
			 remaining_accesses, verification_token, verification_token_expires_in, goals)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`

	_, err := r.db.Exec(query,
		user.ID,
		user.FirstName,
		user.LastName,
		user.Address,
		user.Password,
		user.MedOk,
		user.Cellphone,
		user.SubType,
		user.Email,
		user.EmailVerified,
		user.ExpiresAt,
		user.RemainingAccesses,
		user.VerificationToken,
		user.VerificationTokenExpiresIn,
		user.Goals,
	)

	return err
}

func (r *UserRepository) Update(user *User) error {
	query := `
		UPDATE users
		SET first_name = $2, last_name = $3, address = $4, password = $5, 
			med_ok = $6, cellphone = $7, sub_type = $8, 
			email = $9, email_verified = $10, expires_at = $11, 
			remaining_accesses = $12, verification_token = $13, 
			verification_token_expires_in = $14, goals = $15
		WHERE id = $1
	`

	_, err := r.db.Exec(query,
		user.ID,
		user.FirstName,
		user.LastName,
		user.Address,
		user.Password,
		user.MedOk,
		user.Cellphone,
		user.SubType,
		user.Email,
		user.EmailVerified,
		user.ExpiresAt,
		user.RemainingAccesses,
		user.VerificationToken,
		user.VerificationTokenExpiresIn,
		user.Goals,
	)

	return err
}

func (r *UserRepository) DecrementAccesses(userID string) error {
	query := `UPDATE users SET remaining_accesses = remaining_accesses - 1 WHERE id = $1 AND remaining_accesses > 0`
	_, err := r.db.Exec(query, userID)
	return err
}

func (r *UserRepository) IncrementAccesses(userID string) error {
	query := `UPDATE users SET remaining_accesses = remaining_accesses + 1 WHERE id = $1`
	_, err := r.db.Exec(query, userID)
	return err
}

func (r *UserRepository) Delete(ids []string) error {
	if len(ids) == 0 {
		return nil
	}

	// Use IN clause with placeholders instead of ANY
	query := `DELETE FROM users WHERE id IN (`
	for i := range ids {
		if i > 0 {
			query += ", "
		}
		query += fmt.Sprintf("$%d", i+1)
	}
	query += `)`

	// Convert []string to []interface{} for Exec
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		args[i] = id
	}

	_, err := r.db.Exec(query, args...)
	return err
}

func (r *UserRepository) IncrementRemainingAccesses(userID string) error {
	query := `UPDATE users SET remaining_accesses = remaining_accesses + 1 WHERE id = $1`
	_, err := r.db.Exec(query, userID)
	return err
}

// AdminRepository handles admin operations
type AdminRepository struct {
	db *sql.DB
}

func NewAdminRepository(db *sql.DB) *AdminRepository {
	return &AdminRepository{db: db}
}

func (r *AdminRepository) GetByEmail(email string) (*Admin, error) {
	query := `
		SELECT id, first_name, last_name, email, password, created_at, updated_at
		FROM admins
		WHERE LOWER(email) = LOWER($1)
	`
	
	var admin Admin
	err := r.db.QueryRow(query, email).Scan(
		&admin.ID,
		&admin.FirstName,
		&admin.LastName,
		&admin.Email,
		&admin.Password,
		&admin.CreatedAt,
		&admin.UpdatedAt,
	)
	
	if err != nil {
		return nil, err
	}
	
	return &admin, nil
}

func (r *AdminRepository) GetByID(id string) (*Admin, error) {
	query := `
		SELECT id, first_name, last_name, email, password, created_at, updated_at
		FROM admins
		WHERE id = $1
	`
	
	var admin Admin
	err := r.db.QueryRow(query, id).Scan(
		&admin.ID,
		&admin.FirstName,
		&admin.LastName,
		&admin.Email,
		&admin.Password,
		&admin.CreatedAt,
		&admin.UpdatedAt,
	)
	
	if err != nil {
		return nil, err
	}
	
	return &admin, nil
}

func (r *AdminRepository) Create(admin *Admin) error {
	query := `
		INSERT INTO admins (id, first_name, last_name, email, password, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	
	_, err := r.db.Exec(query,
		admin.ID,
		admin.FirstName,
		admin.LastName,
		admin.Email,
		admin.Password,
		admin.CreatedAt,
		admin.UpdatedAt,
	)
	
	return err
}
