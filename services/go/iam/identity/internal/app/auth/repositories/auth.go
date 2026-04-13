package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	libDatabase "github.com/tea0112/omnitat/libs/go/database"
	"github.com/tea0112/omnitat/services/go/iam/identity/internal/app/auth/models"
)

type RefreshTokenRepositoryImpl struct {
	db *sql.DB
}

func NewRefreshTokenRepository(db *sql.DB) *RefreshTokenRepositoryImpl {
	return &RefreshTokenRepositoryImpl{db: db}
}

func (r *RefreshTokenRepositoryImpl) Create(ctx context.Context, token *models.RefreshToken) error {
	_, err := r.db.ExecContext(ctx, BuildCreateRefreshTokenSQL(),
		token.ID,
		token.EffectiveFamilyID(),
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

func (r *RefreshTokenRepositoryImpl) FindByTokenHash(ctx context.Context, tokenHash string) (*models.RefreshToken, error) {
	return r.findToken(ctx, r.db, BuildFindRefreshTokenByTokenHashSQL(false), tokenHash)
}

func (r *RefreshTokenRepositoryImpl) RevokeByTokenHash(ctx context.Context, tokenHash string, revokedAt time.Time) error {
	_, err := r.db.ExecContext(ctx, BuildRevokeRefreshTokenByTokenHashSQL(), tokenHash, revokedAt)
	return err
}

func (r *RefreshTokenRepositoryImpl) RevokeFamily(ctx context.Context, familyID uuid.UUID, revokedAt time.Time) error {
	if familyID == uuid.Nil {
		return nil
	}

	_, err := r.db.ExecContext(ctx, BuildRevokeRefreshTokenFamilySQL(), familyID, revokedAt)
	return err
}

func (r *RefreshTokenRepositoryImpl) Rotate(ctx context.Context, currentTokenHash string, newToken *models.RefreshToken, now time.Time) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	currentToken, err := r.findToken(ctx, tx, BuildFindRefreshTokenByTokenHashSQL(true), currentTokenHash)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	if currentToken.IsRevoked() || currentToken.IsExpired(now) {
		_ = tx.Rollback()
		return sql.ErrNoRows
	}

	familyID := currentToken.EffectiveFamilyID()
	newToken.FamilyID = familyID
	currentToken.RevokedAt = &now
	currentToken.LastUsedAt = &now
	currentToken.UpdatedAt = now
	newToken.LastUsedAt = &now

	if _, err := tx.ExecContext(ctx, BuildRotateRefreshTokenCurrentSQL(), currentTokenHash, now); err != nil {
		_ = tx.Rollback()
		return err
	}

	if err := r.createWithQuerier(ctx, tx, newToken); err != nil {
		_ = tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

func (r *RefreshTokenRepositoryImpl) ListSessionsByUserID(ctx context.Context, userID uuid.UUID, now time.Time) ([]*models.SessionInfo, error) {
	rows, err := r.db.QueryContext(ctx, BuildListLatestSessionsByUserIDSQL(), userID, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sessions := []*models.SessionInfo{}
	for rows.Next() {
		session, err := scanSessionInfo(rows)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, session)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return sessions, nil
}

func (r *RefreshTokenRepositoryImpl) createWithQuerier(ctx context.Context, querier execQuerier, token *models.RefreshToken) error {
	_, err := querier.ExecContext(ctx, BuildCreateRefreshTokenSQL(),
		token.ID,
		token.EffectiveFamilyID(),
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

func (r *RefreshTokenRepositoryImpl) findToken(ctx context.Context, querier queryRowQuerier, query string, args ...any) (*models.RefreshToken, error) {
	var token models.RefreshToken
	var revokedAt sql.Null[time.Time]
	var lastUsedAt sql.Null[time.Time]
	var userAgent sql.Null[string]
	var ipAddress sql.Null[string]

	err := querier.QueryRowContext(ctx, query, args...).Scan(
		&token.ID,
		&token.FamilyID,
		&token.UserID,
		&token.TokenHash,
		&token.ExpiresAt,
		&revokedAt,
		&lastUsedAt,
		&userAgent,
		&ipAddress,
		&token.CreatedAt,
		&token.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	token.RevokedAt = libDatabase.NullPtr(revokedAt)
	token.LastUsedAt = libDatabase.NullPtr(lastUsedAt)
	token.UserAgent = libDatabase.NullPtr(userAgent)
	token.IPAddress = libDatabase.NullPtr(ipAddress)

	return &token, nil
}

func scanSessionInfo(scanner interface{ Scan(dest ...any) error }) (*models.SessionInfo, error) {
	var session models.SessionInfo
	var lastUsedAt sql.Null[time.Time]
	var revokedAt sql.Null[time.Time]
	var userAgent sql.Null[string]
	var ipAddress sql.Null[string]

	err := scanner.Scan(
		&session.ID,
		&session.UserID,
		&userAgent,
		&ipAddress,
		&session.CreatedAt,
		&session.UpdatedAt,
		&lastUsedAt,
		&session.ExpiresAt,
		&revokedAt,
	)
	if err != nil {
		return nil, err
	}

	session.UserAgent = libDatabase.NullPtr(userAgent)
	session.IPAddress = libDatabase.NullPtr(ipAddress)
	session.LastUsedAt = libDatabase.NullPtr(lastUsedAt)
	session.RevokedAt = libDatabase.NullPtr(revokedAt)

	return &session, nil
}

type queryRowQuerier interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type execQuerier interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}
