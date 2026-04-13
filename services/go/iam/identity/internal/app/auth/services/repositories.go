package services

import (
	"context"
	"time"

	"github.com/google/uuid"
	authModels "github.com/tea0112/omnitat/services/go/iam/identity/internal/app/auth/models"
	"github.com/tea0112/omnitat/services/go/iam/identity/internal/app/users/models"
)

type UserStore interface {
	FindByEmail(ctx context.Context, email string) (*models.User, error)
	FindByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	CreateUser(ctx context.Context, user *models.User) error
}

type RefreshTokenStore interface {
	Create(ctx context.Context, token *authModels.RefreshToken) error
	FindByTokenHash(ctx context.Context, tokenHash string) (*authModels.RefreshToken, error)
	RevokeByTokenHash(ctx context.Context, tokenHash string, revokedAt time.Time) error
	Rotate(ctx context.Context, currentTokenHash string, newToken *authModels.RefreshToken, now time.Time) error
}
