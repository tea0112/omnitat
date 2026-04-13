package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/tea0112/omnitat/services/go/iam/identity/internal/app/auth/models"
)

type RefreshTokenRepositoryImpl struct {
	redisClient *redis.Client
}

func NewRefreshTokenRepository(redisClient *redis.Client) *RefreshTokenRepositoryImpl {
	return &RefreshTokenRepositoryImpl{redisClient: redisClient}
}

func (r *RefreshTokenRepositoryImpl) Create(ctx context.Context, token *models.RefreshToken) error {
	payload, err := json.Marshal(token)
	if err != nil {
		return fmt.Errorf("marshal refresh token: %w", err)
	}

	familyID := token.EffectiveFamilyID()
	_, err = r.redisClient.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		pipe.Set(ctx, refreshTokenKey(token.TokenHash), payload, refreshTokenTTL(token))
		pipe.SAdd(ctx, refreshTokenFamilyKey(familyID), token.TokenHash)
		pipe.SAdd(ctx, refreshTokenUserKey(token.UserID), familyID.String())
		pipe.Expire(ctx, refreshTokenFamilyKey(familyID), refreshTokenTTL(token))
		return nil
	})

	return err
}

func (r *RefreshTokenRepositoryImpl) FindByTokenHash(ctx context.Context, tokenHash string) (*models.RefreshToken, error) {
	payload, err := r.redisClient.Get(ctx, refreshTokenKey(tokenHash)).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}

	var token models.RefreshToken
	if err := json.Unmarshal(payload, &token); err != nil {
		return nil, fmt.Errorf("unmarshal refresh token: %w", err)
	}

	return &token, nil
}

func (r *RefreshTokenRepositoryImpl) RevokeByTokenHash(ctx context.Context, tokenHash string, revokedAt time.Time) error {
	token, err := r.FindByTokenHash(ctx, tokenHash)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return err
	}

	token.RevokedAt = &revokedAt
	token.UpdatedAt = revokedAt

	payload, err := json.Marshal(token)
	if err != nil {
		return fmt.Errorf("marshal refresh token: %w", err)
	}

	return r.redisClient.Set(ctx, refreshTokenKey(token.TokenHash), payload, refreshTokenTTL(token)).Err()
}

func (r *RefreshTokenRepositoryImpl) RevokeFamily(ctx context.Context, familyID uuid.UUID, revokedAt time.Time) error {
	if familyID == uuid.Nil {
		return nil
	}

	familyKey := refreshTokenFamilyKey(familyID)
	tokenHashes, err := r.redisClient.SMembers(ctx, familyKey).Result()
	if err != nil {
		if err == redis.Nil {
			return nil
		}
		return err
	}

	for _, tokenHash := range tokenHashes {
		token, err := r.FindByTokenHash(ctx, tokenHash)
		if err != nil {
			if err == sql.ErrNoRows {
				continue
			}
			return err
		}

		token.RevokedAt = &revokedAt
		token.UpdatedAt = revokedAt

		payload, err := json.Marshal(token)
		if err != nil {
			return fmt.Errorf("marshal refresh token: %w", err)
		}

		if err := r.redisClient.Set(ctx, refreshTokenKey(token.TokenHash), payload, refreshTokenTTL(token)).Err(); err != nil {
			return err
		}
	}

	return nil
}

func (r *RefreshTokenRepositoryImpl) Rotate(ctx context.Context, currentTokenHash string, newToken *models.RefreshToken, now time.Time) error {
	currentKey := refreshTokenKey(currentTokenHash)

	err := r.redisClient.Watch(ctx, func(tx *redis.Tx) error {
		payload, err := tx.Get(ctx, currentKey).Bytes()
		if err != nil {
			if err == redis.Nil {
				return sql.ErrNoRows
			}
			return err
		}

		var currentToken models.RefreshToken
		if err := json.Unmarshal(payload, &currentToken); err != nil {
			return fmt.Errorf("unmarshal refresh token: %w", err)
		}

		familyID := currentToken.EffectiveFamilyID()
		newToken.FamilyID = familyID
		currentToken.RevokedAt = &now
		currentToken.LastUsedAt = &now
		currentToken.UpdatedAt = now
		newToken.LastUsedAt = &now

		currentPayload, err := json.Marshal(&currentToken)
		if err != nil {
			return fmt.Errorf("marshal refresh token: %w", err)
		}
		newPayload, err := json.Marshal(newToken)
		if err != nil {
			return fmt.Errorf("marshal refresh token: %w", err)
		}

		_, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
			pipe.Set(ctx, currentKey, currentPayload, refreshTokenTTL(&currentToken))
			pipe.Set(ctx, refreshTokenKey(newToken.TokenHash), newPayload, refreshTokenTTL(newToken))
			pipe.SAdd(ctx, refreshTokenFamilyKey(familyID), currentToken.TokenHash, newToken.TokenHash)
			pipe.SAdd(ctx, refreshTokenUserKey(newToken.UserID), familyID.String())
			pipe.Expire(ctx, refreshTokenFamilyKey(familyID), refreshTokenTTL(newToken))
			return nil
		})

		return err
	}, currentKey)

	return err
}

func (r *RefreshTokenRepositoryImpl) ListSessionsByUserID(ctx context.Context, userID uuid.UUID, now time.Time) ([]*models.SessionInfo, error) {
	familyIDs, err := r.redisClient.SMembers(ctx, refreshTokenUserKey(userID)).Result()
	if err != nil {
		if err == redis.Nil {
			return []*models.SessionInfo{}, nil
		}
		return nil, err
	}

	sessions := make([]*models.SessionInfo, 0, len(familyIDs))
	for _, familyIDStr := range familyIDs {
		familyID, err := uuid.Parse(familyIDStr)
		if err != nil {
			continue
		}

		session, err := r.loadLatestSessionInFamily(ctx, familyID, now)
		if err != nil {
			return nil, err
		}
		if session == nil {
			continue
		}

		sessions = append(sessions, session)
	}

	return sessions, nil
}

func (r *RefreshTokenRepositoryImpl) loadLatestSessionInFamily(ctx context.Context, familyID uuid.UUID, now time.Time) (*models.SessionInfo, error) {
	tokenHashes, err := r.redisClient.SMembers(ctx, refreshTokenFamilyKey(familyID)).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}

	var latest *models.RefreshToken
	for _, tokenHash := range tokenHashes {
		token, err := r.FindByTokenHash(ctx, tokenHash)
		if err != nil {
			if err == sql.ErrNoRows {
				continue
			}
			return nil, err
		}
		if token.IsExpired(now) {
			continue
		}
		if latest == nil || token.CreatedAt.After(latest.CreatedAt) {
			latest = token
		}
	}

	if latest == nil {
		return nil, nil
	}

	return &models.SessionInfo{
		ID:         latest.EffectiveFamilyID(),
		UserID:     latest.UserID,
		UserAgent:  latest.UserAgent,
		IPAddress:  latest.IPAddress,
		CreatedAt:  latest.CreatedAt,
		UpdatedAt:  latest.UpdatedAt,
		LastUsedAt: latest.LastUsedAt,
		ExpiresAt:  latest.ExpiresAt,
		RevokedAt:  latest.RevokedAt,
	}, nil
}

func refreshTokenKey(tokenHash string) string {
	return "identity:auth:refresh_token:" + tokenHash
}

func refreshTokenFamilyKey(familyID uuid.UUID) string {
	return "identity:auth:refresh_token_family:" + familyID.String()
}

func refreshTokenUserKey(userID uuid.UUID) string {
	return "identity:auth:user_sessions:" + userID.String()
}

func refreshTokenTTL(token *models.RefreshToken) time.Duration {
	ttl := time.Until(token.ExpiresAt)
	if ttl <= 0 {
		return time.Second
	}

	return ttl
}
