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
