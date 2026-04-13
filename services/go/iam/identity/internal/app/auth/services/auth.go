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
	"github.com/tea0112/omnitat/libs/go/security"
	"github.com/tea0112/omnitat/services/go/iam/identity/internal/app/auth/domains"
	authModels "github.com/tea0112/omnitat/services/go/iam/identity/internal/app/auth/models"
	"github.com/tea0112/omnitat/services/go/iam/identity/internal/app/auth/transport/http/dto"
	userModels "github.com/tea0112/omnitat/services/go/iam/identity/internal/app/users/models"
	"github.com/tea0112/omnitat/services/go/iam/identity/pkg/apperrors"
)

type AuthServiceImpl struct {
	clock             datetime.Clock
	userStore         UserStore
	refreshTokenStore RefreshTokenStore
	jwtIssuer         string
	jwtAccessSecret   []byte
	accessTokenTTL    time.Duration
	refreshTokenTTL   time.Duration
}

type TokenConfig struct {
	JWTIssuer       string
	JWTAccessSecret string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
}

func NewAuthService(
	userStore UserStore,
	refreshTokenStore RefreshTokenStore,
	clock datetime.Clock,
	tokenConfig TokenConfig,
) *AuthServiceImpl {
	return &AuthServiceImpl{
		clock:             clock,
		userStore:         userStore,
		refreshTokenStore: refreshTokenStore,
		jwtIssuer:         tokenConfig.JWTIssuer,
		jwtAccessSecret:   []byte(tokenConfig.JWTAccessSecret),
		accessTokenTTL:    tokenConfig.AccessTokenTTL,
		refreshTokenTTL:   tokenConfig.RefreshTokenTTL,
	}
}

func (s *AuthServiceImpl) SignIn(ctx context.Context, signInRequestDTO dto.SignInRequestDTO) (*domains.Session, error) {
	normalizedEmail := normalizeEmail(signInRequestDTO.Email)

	userModel, err := s.userStore.FindByEmail(ctx, normalizedEmail)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperrors.ErrInvalidCredentials
		}

		slog.Error(err.Error())
		return nil, err
	}

	if userModel == nil {
		return nil, apperrors.ErrInvalidCredentials
	}

	accountDomain := domains.NewAccount(userModel.PasswordHash)
	err = accountDomain.VerifyPassword(signInRequestDTO.Password)
	if err != nil {
		slog.Error(err.Error())
		return nil, err
	}

	if !userModel.IsActive {
		return nil, apperrors.ErrUserInactive
	}

	slog.Info("signin success", "user_id", userModel.Id.String(), "email", normalizedEmail)

	return s.issueSession(ctx, userModel, s.clock.Now())
}

func (s *AuthServiceImpl) SignUp(ctx context.Context, signUpRequestDTO dto.SignUpRequestDTO) (*domains.Session, error) {
	now := s.clock.Now()
	user, err := userModels.NewUser(now)
	if err != nil {
		slog.Error(err.Error())
		return nil, err
	}

	normalizedEmail := normalizeEmail(signUpRequestDTO.Email)
	firstName := strings.TrimSpace(signUpRequestDTO.FirstName)
	lastName := strings.TrimSpace(signUpRequestDTO.LastName)

	user.Email = &normalizedEmail
	user.FirstName = &firstName
	if lastName != "" {
		user.LastName = &lastName
	}

	passwordHash, err := security.HashPassword(signUpRequestDTO.Password)
	if err != nil {
		return nil, err
	}
	user.PasswordHash = &passwordHash

	err = s.userStore.CreateUser(ctx, user)
	if err != nil {
		slog.Error("failed to create user: " + err.Error())
		if isDuplicateKeyError(err) {
			return nil, apperrors.ErrEmailAlreadyExists
		}
		return nil, err
	}

	slog.Info("signup success", "user_id", user.Id.String(), "email", normalizedEmail)

	return s.issueSession(ctx, user, now)
}

func (s *AuthServiceImpl) Refresh(ctx context.Context, refreshRequestDTO dto.RefreshRequestDTO) (*domains.TokenPair, error) {
	now := s.clock.Now()
	tokenHash := hashToken(refreshRequestDTO.RefreshToken)

	storedToken, err := s.refreshTokenStore.FindByTokenHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperrors.ErrInvalidRefreshToken
		}
		return nil, err
	}

	if storedToken.IsRevoked() || storedToken.IsExpired(now) {
		return nil, apperrors.ErrInvalidRefreshToken
	}

	userModel, err := s.userStore.FindByID(ctx, storedToken.UserID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperrors.ErrInvalidRefreshToken
		}
		return nil, err
	}

	if userModel == nil {
		return nil, apperrors.ErrInvalidRefreshToken
	}

	if !userModel.IsActive {
		return nil, apperrors.ErrUserInactive
	}

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

	err = s.refreshTokenStore.Rotate(ctx, tokenHash, refreshToken, now)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperrors.ErrInvalidRefreshToken
		}
		slog.Error("failed to rotate refresh token", "user_id", userModel.Id.String(), "error", err.Error())
		return nil, fmt.Errorf("rotate refresh token: %w", err)
	}

	slog.Info("refresh success", "user_id", userModel.Id.String())

	return &domains.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: rawRefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int(s.accessTokenTTL.Seconds()),
	}, nil
}

func (s *AuthServiceImpl) Logout(ctx context.Context, logoutRequestDTO dto.LogoutRequestDTO) error {
	err := s.refreshTokenStore.RevokeByTokenHash(ctx, hashToken(logoutRequestDTO.RefreshToken), s.clock.Now())
	if err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}

	return nil
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

func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}

	return strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique constraint")
}

func (s *AuthServiceImpl) issueSession(ctx context.Context, userModel *userModels.User, now time.Time) (*domains.Session, error) {
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

	err = s.refreshTokenStore.Create(ctx, refreshToken)
	if err != nil {
		slog.Error("failed to persist refresh token", "user_id", userModel.Id.String(), "error", err.Error())
		return nil, fmt.Errorf("persist refresh token: %w", err)
	}

	userDomain := domains.ModelUserToDomain(userModel)
	if userDomain == nil {
		slog.Error(apperrors.ErrMapNilUserDomain.Error())
		return nil, apperrors.ErrMapNilUserDomain
	}

	return &domains.Session{
		AccessToken:  accessToken,
		RefreshToken: rawRefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int(s.accessTokenTTL.Seconds()),
		User:         *userDomain,
	}, nil
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
