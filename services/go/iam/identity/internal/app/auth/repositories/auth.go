package repositories

import (
	"context"
	"database/sql"

	"github.com/tea0112/omnitat/services/go/iam/identity/internal/app/auth/models"
)

type RefreshTokenRepositoryImpl struct {
	db *sql.DB
}

func NewRefreshTokenRepository(db *sql.DB) *RefreshTokenRepositoryImpl {
	return &RefreshTokenRepositoryImpl{db: db}
}

func (r *RefreshTokenRepositoryImpl) Create(ctx context.Context, token *models.RefreshToken) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO refresh_tokens (
			id,
			user_id,
			token_hash,
			expires_at,
			revoked_at,
			last_used_at,
			user_agent,
			ip_address,
			created_at,
			updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`,
		token.ID,
		token.UserID,
		token.TokenHash,
		token.ExpiresAt,
		token.RevokedAt,
		token.LastUsedAt,
		token.UserAgent,
		token.IPAddress,
		token.CreatedAt,
		token.UpdatedAt,
	)

	return err
}
