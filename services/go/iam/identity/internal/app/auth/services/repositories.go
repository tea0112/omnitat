package services

import (
	"context"

	"github.com/google/uuid"
	authModels "github.com/tea0112/omnitat/services/go/iam/identity/internal/app/auth/models"
	"github.com/tea0112/omnitat/services/go/iam/identity/internal/app/users/models"
)

type UserReader interface {
	FindByEmail(ctx context.Context, email string) (*models.User, error)
	FindByID(ctx context.Context, id uuid.UUID) (*models.User, error)
}

type RefreshTokenWriter interface {
	Create(ctx context.Context, token *authModels.RefreshToken) error
}
