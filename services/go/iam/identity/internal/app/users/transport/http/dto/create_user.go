package dto

type CreateUserDTO struct {
	Email     string `json:"email" validate:"required,email"`
	Password  string `json:"password"`
	FirstName string `json:"first_name" validate:"required"`
	LastName  string `json:"last_name"`
}
