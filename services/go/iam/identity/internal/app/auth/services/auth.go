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
	"github.com/google/uuid"
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

	return s.issueSession(ctx, userModel, s.clock.Now(), signInRequestDTO.UserAgent, signInRequestDTO.IPAddress)
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

	return s.issueSession(ctx, user, now, signUpRequestDTO.UserAgent, signUpRequestDTO.IPAddress)
}

func (s *AuthServiceImpl) Refresh(ctx context.Context, refreshRequestDTO dto.RefreshRequestDTO) (*domains.TokenPair, error) {
	now := s.clock.Now()
	tokenHash := hashToken(refreshRequestDTO.RefreshToken)

	storedToken, err := s.refreshTokenStore.FindByTokenHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			slog.Warn("refresh token rejected", "event", "auth.refresh.rejected", "reason", "not_found", "ip_address", refreshRequestDTO.IPAddress, "user_agent", refreshRequestDTO.UserAgent)
			return nil, apperrors.ErrInvalidRefreshToken
		}
		return nil, err
	}

	if storedToken.IsRevoked() {
		slog.Warn("refresh token replay detected", "event", "auth.refresh.replay_detected", "user_id", storedToken.UserID.String(), "family_id", storedToken.EffectiveFamilyID().String(), "ip_address", refreshRequestDTO.IPAddress, "user_agent", refreshRequestDTO.UserAgent)
		if err := s.refreshTokenStore.RevokeFamily(ctx, storedToken.EffectiveFamilyID(), now); err != nil {
			slog.Error("failed to revoke refresh token family after replay", "family_id", storedToken.EffectiveFamilyID().String(), "error", err.Error())
			return nil, fmt.Errorf("revoke refresh token family: %w", err)
		}

		slog.Warn("refresh token family revoked", "event", "auth.refresh.family_revoked", "reason", "replay_detected", "user_id", storedToken.UserID.String(), "family_id", storedToken.EffectiveFamilyID().String(), "revoked_at", now, "ip_address", refreshRequestDTO.IPAddress, "user_agent", refreshRequestDTO.UserAgent)

		return nil, apperrors.ErrInvalidRefreshToken
	}

	if storedToken.IsExpired(now) {
		slog.Warn("refresh token rejected", "event", "auth.refresh.rejected", "reason", "expired", "user_id", storedToken.UserID.String(), "family_id", storedToken.EffectiveFamilyID().String(), "ip_address", refreshRequestDTO.IPAddress, "user_agent", refreshRequestDTO.UserAgent)
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
	applyRefreshTokenMetadata(refreshToken, refreshRequestDTO.UserAgent, refreshRequestDTO.IPAddress)

	err = s.refreshTokenStore.Rotate(ctx, tokenHash, refreshToken, now)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			slog.Warn("refresh token rejected", "event", "auth.refresh.rejected", "reason", "rotate_target_missing", "user_id", userModel.Id.String(), "ip_address", refreshRequestDTO.IPAddress, "user_agent", refreshRequestDTO.UserAgent)
			return nil, apperrors.ErrInvalidRefreshToken
		}
		slog.Error("failed to rotate refresh token", "user_id", userModel.Id.String(), "error", err.Error())
		return nil, fmt.Errorf("rotate refresh token: %w", err)
	}

	slog.Info("refresh token rotated", "event", "auth.refresh.rotated", "user_id", userModel.Id.String(), "family_id", refreshToken.EffectiveFamilyID().String(), "ip_address", refreshRequestDTO.IPAddress, "user_agent", refreshRequestDTO.UserAgent)

	slog.Info("refresh success", "user_id", userModel.Id.String())

	return &domains.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: rawRefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int(s.accessTokenTTL.Seconds()),
	}, nil
}

func (s *AuthServiceImpl) Logout(ctx context.Context, logoutRequestDTO dto.LogoutRequestDTO) error {
	now := s.clock.Now()
	tokenHash := hashToken(logoutRequestDTO.RefreshToken)
	storedToken, err := s.refreshTokenStore.FindByTokenHash(ctx, tokenHash)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("find refresh token for logout: %w", err)
	}

	tokenFound := err == nil && storedToken != nil

	err = s.refreshTokenStore.RevokeByTokenHash(ctx, tokenHash, now)
	if err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}

	if !tokenFound {
		slog.Warn("logout requested for unknown refresh token", "event", "auth.logout.revoke_requested", "reason", "not_found")
		return nil
	}

	slog.Info("refresh token revoked on logout", "event", "auth.logout.refresh_revoked", "user_id", storedToken.UserID.String(), "family_id", storedToken.EffectiveFamilyID().String(), "revoked_at", now)

	return nil
}

func (s *AuthServiceImpl) ListSessions(ctx context.Context, userID uuid.UUID) ([]domains.SessionInfo, error) {
	sessions, err := s.refreshTokenStore.ListSessionsByUserID(ctx, userID, s.clock.Now())
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}

	result := make([]domains.SessionInfo, 0, len(sessions))
	for _, session := range sessions {
		if session == nil {
			continue
		}

		result = append(result, domains.SessionInfo{
			ID:         session.ID,
			UserAgent:  derefString(session.UserAgent),
			IPAddress:  derefString(session.IPAddress),
			CreatedAt:  session.CreatedAt,
			UpdatedAt:  session.UpdatedAt,
			LastUsedAt: session.LastUsedAt,
			ExpiresAt:  session.ExpiresAt,
			RevokedAt:  session.RevokedAt,
		})
	}

	return result, nil
}

func (s *AuthServiceImpl) RevokeSession(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID) error {
	sessions, err := s.refreshTokenStore.ListSessionsByUserID(ctx, userID, s.clock.Now())
	if err != nil {
		return fmt.Errorf("list sessions: %w", err)
	}

	for _, session := range sessions {
		if session != nil && session.ID == sessionID {
			return s.refreshTokenStore.RevokeFamily(ctx, sessionID, s.clock.Now())
		}
	}

	return apperrors.ErrSessionNotFound
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

func (s *AuthServiceImpl) issueSession(ctx context.Context, userModel *userModels.User, now time.Time, userAgent, ipAddress string) (*domains.Session, error) {
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
	applyRefreshTokenMetadata(refreshToken, userAgent, ipAddress)

	err = s.refreshTokenStore.Create(ctx, refreshToken)
	if err != nil {
		slog.Error("failed to persist refresh token", "user_id", userModel.Id.String(), "error", err.Error())
		return nil, fmt.Errorf("persist refresh token: %w", err)
	}

	slog.Info("refresh token issued", "event", "auth.refresh.issued", "user_id", userModel.Id.String(), "family_id", refreshToken.EffectiveFamilyID().String(), "ip_address", ipAddress, "user_agent", userAgent)

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

func applyRefreshTokenMetadata(token *authModels.RefreshToken, userAgent, ipAddress string) {
	if token == nil {
		return
	}

	userAgent = strings.TrimSpace(userAgent)
	ipAddress = strings.TrimSpace(ipAddress)

	if userAgent != "" {
		token.UserAgent = &userAgent
	}

	if ipAddress != "" {
		token.IPAddress = &ipAddress
	}
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}

	return *value
}
