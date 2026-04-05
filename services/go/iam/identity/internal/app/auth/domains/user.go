package domains

import (
	"time"

	"github.com/google/uuid"
	"github.com/tea0112/omnitat/services/go/iam/identity/internal/app/users/models"
)

type User struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func ModelUserToDomain(m *models.User) *User {
	if m == nil {
		return nil
	}
	email := ""
	if m.Email != nil {
		email = *m.Email
	}
	firstName := ""
	if m.FirstName != nil {
		firstName = *m.FirstName
	}
	lastName := ""
	if m.LastName != nil {
		lastName = *m.LastName
	}
	return &User{
		ID:        m.Id,
		Email:     email,
		FirstName: firstName,
		LastName:  lastName,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}
