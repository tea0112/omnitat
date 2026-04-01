package services

import (
	"context"

	"github.com/tea0112/omnitat/services/go/iam/identity/internal/app/users/models"
)

type UserRepository interface {
	CreateUser(ctx context.Context, user *models.User) error
}