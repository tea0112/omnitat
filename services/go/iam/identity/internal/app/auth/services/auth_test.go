package services

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/tea0112/omnitat/services/go/iam/identity/internal/app/auth/models"
	"github.com/tea0112/omnitat/services/go/iam/identity/internal/app/auth/transport/http/dto"
	userModels "github.com/tea0112/omnitat/services/go/iam/identity/internal/app/users/models"
	"github.com/tea0112/omnitat/services/go/iam/identity/pkg/apperrors"
)

type fakeClock struct {
	now time.Time
}

func (c *fakeClock) Now() time.Time {
	return c.now
}

type fakeUserStore struct {
	byEmail map[string]*userModels.User
	byID    map[uuid.UUID]*userModels.User
	err     error
}

func newFakeUserStore(users ...*userModels.User) *fakeUserStore {
	store := &fakeUserStore{
		byEmail: map[string]*userModels.User{},
		byID:    map[uuid.UUID]*userModels.User{},
	}
	for _, user := range users {
		if user == nil {
			continue
		}
		store.byID[user.Id] = user
		if user.Email != nil {
			store.byEmail[*user.Email] = user
		}
	}

	return store
}

func (s *fakeUserStore) FindByEmail(_ context.Context, email string) (*userModels.User, error) {
	if s.err != nil {
		return nil, s.err
	}
	user, ok := s.byEmail[email]
	if !ok {
		return nil, sql.ErrNoRows
	}
	return user, nil
}

func (s *fakeUserStore) FindByID(_ context.Context, id uuid.UUID) (*userModels.User, error) {
	if s.err != nil {
		return nil, s.err
	}
	user, ok := s.byID[id]
	if !ok {
		return nil, sql.ErrNoRows
	}
	return user, nil
}

func (s *fakeUserStore) CreateUser(_ context.Context, user *userModels.User) error {
	if s.err != nil {
		return s.err
	}
	s.byID[user.Id] = user
	if user.Email != nil {
		s.byEmail[*user.Email] = user
	}
	return nil
}

type fakeRefreshTokenStore struct {
	tokens map[string]*models.RefreshToken
	err    error
}

func newFakeRefreshTokenStore(tokens ...*models.RefreshToken) *fakeRefreshTokenStore {
	store := &fakeRefreshTokenStore{tokens: map[string]*models.RefreshToken{}}
	for _, token := range tokens {
		store.tokens[token.TokenHash] = token
	}
	return store
}

func (s *fakeRefreshTokenStore) Create(_ context.Context, token *models.RefreshToken) error {
	if s.err != nil {
		return s.err
	}
	s.tokens[token.TokenHash] = token
	return nil
}

func (s *fakeRefreshTokenStore) FindByTokenHash(_ context.Context, tokenHash string) (*models.RefreshToken, error) {
	if s.err != nil {
		return nil, s.err
	}
	token, ok := s.tokens[tokenHash]
	if !ok {
		return nil, sql.ErrNoRows
	}
	return token, nil
}

func (s *fakeRefreshTokenStore) ListSessionsByUserID(_ context.Context, userID uuid.UUID, now time.Time) ([]*models.SessionInfo, error) {
	if s.err != nil {
		return nil, s.err
	}

	byFamily := map[uuid.UUID]*models.RefreshToken{}
	for _, token := range s.tokens {
		if token.UserID != userID || token.IsExpired(now) {
			continue
		}
		familyID := token.EffectiveFamilyID()
		current, ok := byFamily[familyID]
		if !ok || token.CreatedAt.After(current.CreatedAt) {
			byFamily[familyID] = token
		}
	}

	sessions := make([]*models.SessionInfo, 0, len(byFamily))
	for familyID, token := range byFamily {
		sessions = append(sessions, &models.SessionInfo{
			ID:         familyID,
			UserID:     token.UserID,
			UserAgent:  token.UserAgent,
			IPAddress:  token.IPAddress,
			CreatedAt:  token.CreatedAt,
			UpdatedAt:  token.UpdatedAt,
			LastUsedAt: token.LastUsedAt,
			ExpiresAt:  token.ExpiresAt,
			RevokedAt:  token.RevokedAt,
		})
	}

	return sessions, nil
}

func (s *fakeRefreshTokenStore) RevokeByTokenHash(_ context.Context, tokenHash string, _ time.Time) error {
	if s.err != nil {
		return s.err
	}
	token, ok := s.tokens[tokenHash]
	if !ok {
		return nil
	}
	now := time.Now().UTC()
	token.RevokedAt = &now
	token.UpdatedAt = now
	return nil
}

func (s *fakeRefreshTokenStore) RevokeFamily(_ context.Context, familyID uuid.UUID, _ time.Time) error {
	if s.err != nil {
		return s.err
	}
	now := time.Now().UTC()
	for _, token := range s.tokens {
		if token.EffectiveFamilyID() != familyID {
			continue
		}
		token.RevokedAt = &now
		token.UpdatedAt = now
	}
	return nil
}

func (s *fakeRefreshTokenStore) Rotate(_ context.Context, currentTokenHash string, newToken *models.RefreshToken, _ time.Time) error {
	if s.err != nil {
		return s.err
	}
	currentToken, ok := s.tokens[currentTokenHash]
	if !ok {
		return sql.ErrNoRows
	}
	familyID := currentToken.EffectiveFamilyID()
	newToken.FamilyID = familyID
	now := time.Now().UTC()
	currentToken.RevokedAt = &now
	currentToken.UpdatedAt = now
	s.tokens[newToken.TokenHash] = newToken
	return nil
}

func TestSignUpNormalizesEmailAndStoresRefreshToken(t *testing.T) {
	now := time.Date(2026, 4, 13, 12, 0, 0, 0, time.UTC)
	userStore := newFakeUserStore()
	refreshStore := newFakeRefreshTokenStore()
	service := NewAuthService(userStore, refreshStore, &fakeClock{now: now}, TokenConfig{
		JWTIssuer:       "identity-service",
		JWTAccessSecret: "test-secret",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 24 * time.Hour,
	})

	session, err := service.SignUp(context.Background(), dto.SignUpRequestDTO{
		Email:     " Test@Example.COM ",
		Password:  "StrongPass123!",
		FirstName: "John",
		LastName:  "Doe",
		UserAgent: "test-agent",
		IPAddress: "203.0.113.10",
	})
	if err != nil {
		t.Fatalf("SignUp() error = %v", err)
	}
	if session.User.Email != "test@example.com" {
		t.Fatalf("expected normalized email, got %q", session.User.Email)
	}
	if session.RefreshToken == "" {
		t.Fatal("expected refresh token to be returned")
	}
	if len(refreshStore.tokens) != 1 {
		t.Fatalf("expected 1 stored refresh token, got %d", len(refreshStore.tokens))
	}
	for _, token := range refreshStore.tokens {
		if token.UserAgent == nil || *token.UserAgent != "test-agent" {
			t.Fatal("expected refresh token user agent to be stored")
		}
		if token.IPAddress == nil || *token.IPAddress != "203.0.113.10" {
			t.Fatal("expected refresh token ip address to be stored")
		}
	}
	storedUser, err := userStore.FindByEmail(context.Background(), "test@example.com")
	if err != nil {
		t.Fatalf("stored user lookup error = %v", err)
	}
	if storedUser.PasswordHash == nil || *storedUser.PasswordHash == "" {
		t.Fatal("expected password hash to be stored")
	}
	if *storedUser.PasswordHash == "StrongPass123!" {
		t.Fatal("expected password to be hashed")
	}
}

func TestRefreshRotatesRefreshToken(t *testing.T) {
	now := time.Date(2026, 4, 13, 12, 0, 0, 0, time.UTC)
	email := "user@example.com"
	user := &userModels.User{Id: uuid.MustParse("01962da8-90b6-7a1a-b5f3-123456789012"), Email: &email, IsActive: true, CreatedAt: now, UpdatedAt: now}
	currentRaw := "current-refresh-token"
	currentHash := hashToken(currentRaw)
	refreshToken := &models.RefreshToken{ID: uuid.MustParse("01962da8-90b6-7a1a-b5f3-123456789013"), UserID: user.Id, TokenHash: currentHash, ExpiresAt: now.Add(24 * time.Hour), CreatedAt: now, UpdatedAt: now}

	userStore := newFakeUserStore(user)
	refreshStore := newFakeRefreshTokenStore(refreshToken)
	service := NewAuthService(userStore, refreshStore, &fakeClock{now: now}, TokenConfig{
		JWTIssuer:       "identity-service",
		JWTAccessSecret: "test-secret",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 24 * time.Hour,
	})

	pair, err := service.Refresh(context.Background(), dto.RefreshRequestDTO{RefreshToken: currentRaw, UserAgent: "rotated-agent", IPAddress: "198.51.100.20"})
	if err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}
	if pair.RefreshToken == "" || pair.RefreshToken == currentRaw {
		t.Fatal("expected a new refresh token to be returned")
	}
	if refreshStore.tokens[currentHash].RevokedAt == nil {
		t.Fatal("expected old refresh token to be revoked")
	}
	if _, ok := refreshStore.tokens[hashToken(pair.RefreshToken)]; !ok {
		t.Fatal("expected new refresh token to be stored")
	}
	rotatedToken := refreshStore.tokens[hashToken(pair.RefreshToken)]
	if rotatedToken.UserAgent == nil || *rotatedToken.UserAgent != "rotated-agent" {
		t.Fatal("expected rotated token user agent to be updated")
	}
	if rotatedToken.IPAddress == nil || *rotatedToken.IPAddress != "198.51.100.20" {
		t.Fatal("expected rotated token ip address to be updated")
	}
}

func TestLogoutRevokesRefreshToken(t *testing.T) {
	now := time.Date(2026, 4, 13, 12, 0, 0, 0, time.UTC)
	currentRaw := "logout-refresh-token"
	currentHash := hashToken(currentRaw)
	refreshStore := newFakeRefreshTokenStore(&models.RefreshToken{TokenHash: currentHash, ExpiresAt: now.Add(time.Hour)})
	service := NewAuthService(newFakeUserStore(), refreshStore, &fakeClock{now: now}, TokenConfig{
		JWTIssuer:       "identity-service",
		JWTAccessSecret: "test-secret",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 24 * time.Hour,
	})

	if err := service.Logout(context.Background(), dto.LogoutRequestDTO{RefreshToken: currentRaw}); err != nil {
		t.Fatalf("Logout() error = %v", err)
	}
	if refreshStore.tokens[currentHash].RevokedAt == nil {
		t.Fatal("expected refresh token to be revoked")
	}
}

func TestRefreshReuseRevokesTokenFamily(t *testing.T) {
	now := time.Date(2026, 4, 13, 12, 0, 0, 0, time.UTC)
	email := "user@example.com"
	familyID := uuid.MustParse("01962da8-90b6-7a1a-b5f3-123456789099")
	user := &userModels.User{Id: uuid.MustParse("01962da8-90b6-7a1a-b5f3-123456789012"), Email: &email, IsActive: true, CreatedAt: now, UpdatedAt: now}
	oldRaw := "old-refresh-token"
	activeRaw := "active-refresh-token"
	revokedAt := now.Add(-5 * time.Minute)
	oldToken := &models.RefreshToken{ID: uuid.MustParse("01962da8-90b6-7a1a-b5f3-123456789013"), FamilyID: familyID, UserID: user.Id, TokenHash: hashToken(oldRaw), ExpiresAt: now.Add(24 * time.Hour), RevokedAt: &revokedAt, CreatedAt: now.Add(-10 * time.Minute), UpdatedAt: revokedAt}
	activeToken := &models.RefreshToken{ID: uuid.MustParse("01962da8-90b6-7a1a-b5f3-123456789014"), FamilyID: familyID, UserID: user.Id, TokenHash: hashToken(activeRaw), ExpiresAt: now.Add(24 * time.Hour), CreatedAt: now, UpdatedAt: now}

	userStore := newFakeUserStore(user)
	refreshStore := newFakeRefreshTokenStore(oldToken, activeToken)
	service := NewAuthService(userStore, refreshStore, &fakeClock{now: now}, TokenConfig{
		JWTIssuer:       "identity-service",
		JWTAccessSecret: "test-secret",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 24 * time.Hour,
	})

	_, err := service.Refresh(context.Background(), dto.RefreshRequestDTO{RefreshToken: oldRaw})
	if !errors.Is(err, apperrors.ErrInvalidRefreshToken) {
		t.Fatalf("expected ErrInvalidRefreshToken, got %v", err)
	}
	if refreshStore.tokens[hashToken(activeRaw)].RevokedAt == nil {
		t.Fatal("expected active token to be revoked after replay detection")
	}
}

func TestSignInReturnsInvalidCredentialsForUnknownEmail(t *testing.T) {
	service := NewAuthService(newFakeUserStore(), newFakeRefreshTokenStore(), &fakeClock{now: time.Now().UTC()}, TokenConfig{
		JWTIssuer:       "identity-service",
		JWTAccessSecret: "test-secret",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 24 * time.Hour,
	})

	_, err := service.SignIn(context.Background(), dto.SignInRequestDTO{Email: "missing@example.com", Password: "Secret123!"})
	if !errors.Is(err, apperrors.ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestListSessionsReturnsUserSessions(t *testing.T) {
	now := time.Now().UTC()
	email := "user@example.com"
	user := &userModels.User{Id: uuid.MustParse("01962da8-90b6-7a1a-b5f3-123456789012"), Email: &email, IsActive: true, CreatedAt: now, UpdatedAt: now}
	agent := "session-agent"
	ip := "198.51.100.44"
	familyID := uuid.MustParse("01962da8-90b6-7a1a-b5f3-123456789015")
	refreshToken := &models.RefreshToken{ID: uuid.MustParse("01962da8-90b6-7a1a-b5f3-123456789016"), FamilyID: familyID, UserID: user.Id, TokenHash: hashToken("session-refresh"), UserAgent: &agent, IPAddress: &ip, ExpiresAt: now.Add(24 * time.Hour), CreatedAt: now, UpdatedAt: now}

	userStore := newFakeUserStore(user)
	refreshStore := newFakeRefreshTokenStore(refreshToken)
	service := NewAuthService(userStore, refreshStore, &fakeClock{now: now}, TokenConfig{
		JWTIssuer:       "identity-service",
		JWTAccessSecret: "test-secret",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 24 * time.Hour,
	})

	sessions, err := service.ListSessions(context.Background(), user.Id)
	if err != nil {
		t.Fatalf("ListSessions() error = %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}
	if sessions[0].ID != familyID {
		t.Fatalf("expected session id %s, got %s", familyID, sessions[0].ID)
	}
	if sessions[0].UserAgent != agent || sessions[0].IPAddress != ip {
		t.Fatal("expected session metadata to be returned")
	}
}

func TestRevokeSessionRevokesFamily(t *testing.T) {
	now := time.Now().UTC()
	email := "user@example.com"
	user := &userModels.User{Id: uuid.MustParse("01962da8-90b6-7a1a-b5f3-123456789012"), Email: &email, IsActive: true, CreatedAt: now, UpdatedAt: now}
	familyID := uuid.MustParse("01962da8-90b6-7a1a-b5f3-123456789015")
	first := &models.RefreshToken{ID: uuid.MustParse("01962da8-90b6-7a1a-b5f3-123456789016"), FamilyID: familyID, UserID: user.Id, TokenHash: hashToken("session-refresh-1"), ExpiresAt: now.Add(24 * time.Hour), CreatedAt: now.Add(-time.Minute), UpdatedAt: now.Add(-time.Minute)}
	second := &models.RefreshToken{ID: uuid.MustParse("01962da8-90b6-7a1a-b5f3-123456789017"), FamilyID: familyID, UserID: user.Id, TokenHash: hashToken("session-refresh-2"), ExpiresAt: now.Add(24 * time.Hour), CreatedAt: now, UpdatedAt: now}

	userStore := newFakeUserStore(user)
	refreshStore := newFakeRefreshTokenStore(first, second)
	service := NewAuthService(userStore, refreshStore, &fakeClock{now: now}, TokenConfig{
		JWTIssuer:       "identity-service",
		JWTAccessSecret: "test-secret",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 24 * time.Hour,
	})

	if err := service.RevokeSession(context.Background(), user.Id, familyID); err != nil {
		t.Fatalf("RevokeSession() error = %v", err)
	}
	if refreshStore.tokens[first.TokenHash].RevokedAt == nil || refreshStore.tokens[second.TokenHash].RevokedAt == nil {
		t.Fatal("expected all tokens in session family to be revoked")
	}
}
