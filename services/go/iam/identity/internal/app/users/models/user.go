package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	Id           uuid.UUID  `db:"id" json:"id"`
	Email        *string    `db:"email" json:"email"`
	PasswordHash *string    `db:"password_hash" json:"password_hash"`
	FirstName    *string    `db:"first_name" json:"first_name"`
	LastName     *string    `db:"last_name" json:"last_name"`
	IsActive     bool       `db:"is_active" json:"is_active"`
	CreatedAt    time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time  `db:"updated_at" json:"updated_at"`
	DeletedAt    *time.Time `db:"deleted_at" json:"deleted_at"`
}

func NewUser(now time.Time) (*User, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}

	return &User{
		Id:        id,
		CreatedAt: now,
		UpdatedAt: now,
		IsActive:  true,
	}, nil
}

func UserFields() *UserField {
	return &UserField{
		TableName:          "users",
		IdColumn:           "id",
		EmailColumn:        "email",
		PasswordHashColumn: "password_hash",
		FirstNameColumn:    "first_name",
		LastNameColumn:     "last_name",
		IsActiveColumn:     "is_active",
		CreatedAtColumn:    "created_at",
		UpdatedAtColumn:    "updated_at",
		DeletedAtColumn:    "deleted_at",
	}
}

type UserField struct {
	TableName          string
	IdColumn           string
	EmailColumn        string
	PasswordHashColumn string
	FirstNameColumn    string
	LastNameColumn     string
	IsActiveColumn     string
	CreatedAtColumn    string
	UpdatedAtColumn    string
	DeletedAtColumn    string
}
