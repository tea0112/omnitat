package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/tea0112/omnitat/libs/go/datetime"
	"github.com/tea0112/omnitat/services/go/iam/identity/internal/app/auth/domains"
	authModels "github.com/tea0112/omnitat/services/go/iam/identity/internal/app/auth/models"
	"github.com/tea0112/omnitat/services/go/iam/identity/internal/app/auth/transport/http/dto"
	userModels "github.com/tea0112/omnitat/services/go/iam/identity/internal/app/users/models"
	"github.com/tea0112/omnitat/services/go/iam/identity/pkg/apperrors"
)

type AuthServiceImpl struct {
	clock              datetime.Clock
	userReader         UserReader
	refreshTokenWriter RefreshTokenWriter
	jwtIssuer          string
	jwtAccessSecret    []byte
	accessTokenTTL     time.Duration
	refreshTokenTTL    time.Duration
}

type TokenConfig struct {
	JWTIssuer       string
	JWTAccessSecret string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
}

func NewAuthService(
	userReader UserReader,
	refreshTokenWriter RefreshTokenWriter,
	clock datetime.Clock,
	tokenConfig TokenConfig,
) *AuthServiceImpl {
	return &AuthServiceImpl{
		clock:              clock,
		userReader:         userReader,
		refreshTokenWriter: refreshTokenWriter,
		jwtIssuer:          tokenConfig.JWTIssuer,
		jwtAccessSecret:    []byte(tokenConfig.JWTAccessSecret),
		accessTokenTTL:     tokenConfig.AccessTokenTTL,
		refreshTokenTTL:    tokenConfig.RefreshTokenTTL,
	}
}

func (s *AuthServiceImpl) Login(ctx context.Context, loginRequestDTO dto.LoginRequestDTO) (*domains.Login, error) {
	normalizedEmail := normalizeEmail(loginRequestDTO.Email)

	userModel, err := s.userReader.FindByEmail(ctx, normalizedEmail)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperrors.ErrInvalidCredentials
		}

		slog.Error(err.Error())
		return nil, err
	}

	if userModel == nil {
		slog.Error(apperrors.ErrInvalidCredentials.Error())
		return nil, apperrors.ErrInvalidCredentials
	}

	accountDomain := domains.NewAccount(userModel.PasswordHash)
	err = accountDomain.VerifyPassword(loginRequestDTO.Password)
	if err != nil {
		slog.Error(err.Error())
		return nil, err
	}

	// TODO: consider move to domain
	if !userModel.IsActive {
		return nil, apperrors.ErrUserInactive
	}

	now := s.clock.Now()
	accessToken, err := s.generateAccessToken(userModel, now)
	if err != nil {
		slog.Error(err.Error())
		return nil, fmt.Errorf("generate access token: %w", err)
	}

	rawRefreshToken, err := generateOpaqueToken()
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	refreshToken, err := authModels.NewRefreshToken(
		userModel.Id,
		hashToken(rawRefreshToken),
		now,
		now.Add(s.refreshTokenTTL),
	)
	if err != nil {
		slog.Error(err.Error())
		return nil, fmt.Errorf("new refresh token: %w", err)
	}

	err = s.refreshTokenWriter.Create(ctx, refreshToken)
	if err != nil {
		slog.Error("failed to persist refresh token", "user_id", userModel.Id.String(), "error", err.Error())
		return nil, fmt.Errorf("persist refresh token: %w", err)
	}

	userDomain := domains.ModelUserToDomain(userModel)
	if userDomain == nil {
		slog.Error(apperrors.ErrMapNilUserDomain.Error())
		return nil, apperrors.ErrMapNilUserDomain
	}

	slog.Info("login success", "user_id", userModel.Id.String(), "email", normalizedEmail)

	return &domains.Login{
		AccessToken:  accessToken,
		RefreshToken: rawRefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int(s.accessTokenTTL.Seconds()),
		User:         *userDomain,
	}, nil
}

func (s *AuthServiceImpl) Refresh() error {
	return errors.New("refresh not implemented")
}

func (s *AuthServiceImpl) Logout() error {
	return errors.New("logout not implemented")
}

type accessTokenClaims struct {
	Email string `json:"email"`
	jwt.RegisteredClaims
}

func (s *AuthServiceImpl) generateAccessToken(user *userModels.User, now time.Time) (string, error) {
	claims := accessTokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.jwtIssuer,
			Subject:   user.Id.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.accessTokenTTL)),
		},
	}

	if user.Email != nil {
		claims.Email = *user.Email
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtAccessSecret)
}

func normalizeEmail(email string) string {
	return strings.TrimSpace(strings.ToLower(email))
}

func generateOpaqueToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func hashToken(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
