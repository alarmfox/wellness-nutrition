package models

import (
	"database/sql"
	"time"
)

type Role string

const (
	RoleAdmin Role = "ADMIN"
	RoleUser  Role = "USER"
)

type SubType string

const (
	SubTypeShared SubType = "SHARED"
	SubTypeSingle SubType = "SINGLE"
)

type User struct {
	ID                          string
	FirstName                   string
	LastName                    string
	Address                     string
	Password                    sql.NullString
	Role                        Role
	MedOk                       bool
	Cellphone                   sql.NullString
	SubType                     SubType
	Email                       string
	EmailVerified               sql.NullTime
	ExpiresAt                   time.Time
	RemainingAccesses           int
	VerificationToken           sql.NullString
	VerificationTokenExpiresIn  sql.NullTime
	Goals                       sql.NullString
}

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) GetByEmail(email string) (*User, error) {
	query := `
		SELECT id, "firstName", "lastName", address, password, role, "medOk", 
			   cellphone, "subType", email, "emailVerified", "expiresAt", 
			   "remainingAccesses", "verificationToken", "verificationTokenExpiresIn", goals
		FROM "User"
		WHERE LOWER(email) = LOWER($1)
	`
	
	var user User
	err := r.db.QueryRow(query, email).Scan(
		&user.ID,
		&user.FirstName,
		&user.LastName,
		&user.Address,
		&user.Password,
		&user.Role,
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
		SELECT id, "firstName", "lastName", address, password, role, "medOk", 
			   cellphone, "subType", email, "emailVerified", "expiresAt", 
			   "remainingAccesses", "verificationToken", "verificationTokenExpiresIn", goals
		FROM "User"
		WHERE id = $1
	`
	
	var user User
	err := r.db.QueryRow(query, id).Scan(
		&user.ID,
		&user.FirstName,
		&user.LastName,
		&user.Address,
		&user.Password,
		&user.Role,
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
		SELECT id, "firstName", "lastName", address, password, role, "medOk", 
			   cellphone, "subType", email, "emailVerified", "expiresAt", 
			   "remainingAccesses", "verificationToken", "verificationTokenExpiresIn", goals
		FROM "User"
		WHERE role = $1
		ORDER BY "firstName", "lastName"
	`
	
	rows, err := r.db.Query(query, RoleUser)
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
			&user.Role,
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
		INSERT INTO "User" 
			(id, "firstName", "lastName", address, password, role, "medOk", 
			 cellphone, "subType", email, "emailVerified", "expiresAt", 
			 "remainingAccesses", "verificationToken", "verificationTokenExpiresIn", goals)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
	`
	
	_, err := r.db.Exec(query,
		user.ID,
		user.FirstName,
		user.LastName,
		user.Address,
		user.Password,
		user.Role,
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
		UPDATE "User"
		SET "firstName" = $2, "lastName" = $3, address = $4, password = $5, 
			role = $6, "medOk" = $7, cellphone = $8, "subType" = $9, 
			email = $10, "emailVerified" = $11, "expiresAt" = $12, 
			"remainingAccesses" = $13, "verificationToken" = $14, 
			"verificationTokenExpiresIn" = $15, goals = $16
		WHERE id = $1
	`
	
	_, err := r.db.Exec(query,
		user.ID,
		user.FirstName,
		user.LastName,
		user.Address,
		user.Password,
		user.Role,
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

func (r *UserRepository) Delete(ids []string) error {
	query := `DELETE FROM "User" WHERE id = ANY($1)`
	_, err := r.db.Exec(query, ids)
	return err
}
