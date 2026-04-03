package dto

import (
	"time"

	"github.com/google/uuid"
)

type UserResponseDTO struct {
	Id        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreateUserDTO struct {
	Email     string `json:"email" validate:"required,email"`
	Password  string `json:"password"`
	FirstName string `json:"first_name" validate:"required"`
	LastName  string `json:"last_name"`
}

type ErrorCodes struct {
	INVALID_JSON         string
	INVALID_DATA         string
	INVALID_EMAIL        string
	WEAK_PASSWORD        string
	EMAIL_ALREADY_EXISTS string
	CREATE_FAILED        string
}

var ErrorCode = ErrorCodes{
	INVALID_JSON:         "INVALID_JSON",
	INVALID_DATA:         "INVALID_DATA",
	INVALID_EMAIL:        "INVALID_EMAIL",
	WEAK_PASSWORD:        "WEAK_PASSWORD",
	EMAIL_ALREADY_EXISTS: "EMAIL_ALREADY_EXISTS",
	CREATE_FAILED:        "CREATE_FAILED",
}
