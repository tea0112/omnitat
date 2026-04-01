package http

import (
	"context"

	"github.com/tea0112/omnitat/services/go/iam/identity/internal/app/users/models"
	"github.com/tea0112/omnitat/services/go/iam/identity/internal/app/users/transport/http/dto"
)

type UserService interface {
	CreateUser(ctx context.Context, createUserDTO *dto.CreateUserDTO) (*models.User, error)
}
