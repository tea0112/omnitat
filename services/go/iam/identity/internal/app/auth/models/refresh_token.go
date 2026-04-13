package models

import (
	"time"

	"github.com/google/uuid"
)

type RefreshToken struct {
	ID         uuid.UUID  `db:"id"`
	UserID     uuid.UUID  `db:"user_id"`
	TokenHash  string     `db:"token_hash"`
	ExpiresAt  time.Time  `db:"expires_at"`
	RevokedAt  *time.Time `db:"revoked_at"`
	LastUsedAt *time.Time `db:"last_used_at"`
	UserAgent  *string    `db:"user_agent"`
	IPAddress  *string    `db:"ip_address"`
	CreatedAt  time.Time  `db:"created_at"`
	UpdatedAt  time.Time  `db:"updated_at"`
}

func (t *RefreshToken) IsRevoked() bool {
	return t != nil && t.RevokedAt != nil
}

func (t *RefreshToken) IsExpired(now time.Time) bool {
	if t == nil {
		return true
	}

	return !now.Before(t.ExpiresAt)
}

func NewRefreshToken(userID uuid.UUID, tokenHash string, now, expiresAt time.Time) (*RefreshToken, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}

	return &RefreshToken{
		ID:        id,
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}
