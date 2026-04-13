package models

import (
	"time"

	"github.com/google/uuid"
)

type RefreshToken struct {
	ID         uuid.UUID  `db:"id"`
	FamilyID   uuid.UUID  `db:"family_id"`
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

func RefreshTokenFields() *RefreshTokenField {
	return &RefreshTokenField{
		TableName:        "refresh_tokens",
		IDColumn:         "id",
		FamilyIDColumn:   "family_id",
		UserIDColumn:     "user_id",
		TokenHashColumn:  "token_hash",
		ExpiresAtColumn:  "expires_at",
		RevokedAtColumn:  "revoked_at",
		LastUsedAtColumn: "last_used_at",
		UserAgentColumn:  "user_agent",
		IPAddressColumn:  "ip_address",
		CreatedAtColumn:  "created_at",
		UpdatedAtColumn:  "updated_at",
	}
}

type RefreshTokenField struct {
	TableName        string
	IDColumn         string
	FamilyIDColumn   string
	UserIDColumn     string
	TokenHashColumn  string
	ExpiresAtColumn  string
	RevokedAtColumn  string
	LastUsedAtColumn string
	UserAgentColumn  string
	IPAddressColumn  string
	CreatedAtColumn  string
	UpdatedAtColumn  string
}

type SessionInfo struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	UserAgent  *string
	IPAddress  *string
	CreatedAt  time.Time
	UpdatedAt  time.Time
	LastUsedAt *time.Time
	ExpiresAt  time.Time
	RevokedAt  *time.Time
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
		FamilyID:  id,
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func (t *RefreshToken) EffectiveFamilyID() uuid.UUID {
	if t == nil {
		return uuid.Nil
	}

	if t.FamilyID != uuid.Nil {
		return t.FamilyID
	}

	return t.ID
}
