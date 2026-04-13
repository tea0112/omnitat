package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

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

	return r.redisClient.Set(ctx, refreshTokenKey(token.TokenHash), payload, refreshTokenTTL(token)).Err()
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
	_ = revokedAt
	return r.redisClient.Del(ctx, refreshTokenKey(tokenHash)).Err()
}

func (r *RefreshTokenRepositoryImpl) Rotate(ctx context.Context, currentTokenHash string, newToken *models.RefreshToken, now time.Time) error {
	_ = now

	currentKey := refreshTokenKey(currentTokenHash)
	newPayload, err := json.Marshal(newToken)
	if err != nil {
		return fmt.Errorf("marshal refresh token: %w", err)
	}

	err = r.redisClient.Watch(ctx, func(tx *redis.Tx) error {
		exists, err := tx.Exists(ctx, currentKey).Result()
		if err != nil {
			return err
		}
		if exists == 0 {
			return sql.ErrNoRows
		}

		_, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
			pipe.Del(ctx, currentKey)
			pipe.Set(ctx, refreshTokenKey(newToken.TokenHash), newPayload, refreshTokenTTL(newToken))
			return nil
		})

		return err
	}, currentKey)

	return err
}

func refreshTokenKey(tokenHash string) string {
	return "identity:auth:refresh_token:" + tokenHash
}

func refreshTokenTTL(token *models.RefreshToken) time.Duration {
	ttl := time.Until(token.ExpiresAt)
	if ttl <= 0 {
		return time.Second
	}

	return ttl
}
