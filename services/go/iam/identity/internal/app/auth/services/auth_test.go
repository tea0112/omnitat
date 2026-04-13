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

func (s *fakeRefreshTokenStore) RevokeByTokenHash(_ context.Context, tokenHash string, _ time.Time) error {
	if s.err != nil {
		return s.err
	}
	delete(s.tokens, tokenHash)
	return nil
}

func (s *fakeRefreshTokenStore) Rotate(_ context.Context, currentTokenHash string, newToken *models.RefreshToken, _ time.Time) error {
	if s.err != nil {
		return s.err
	}
	if _, ok := s.tokens[currentTokenHash]; !ok {
		return sql.ErrNoRows
	}
	delete(s.tokens, currentTokenHash)
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

	pair, err := service.Refresh(context.Background(), dto.RefreshRequestDTO{RefreshToken: currentRaw})
	if err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}
	if pair.RefreshToken == "" || pair.RefreshToken == currentRaw {
		t.Fatal("expected a new refresh token to be returned")
	}
	if _, ok := refreshStore.tokens[currentHash]; ok {
		t.Fatal("expected old refresh token to be removed")
	}
	if _, ok := refreshStore.tokens[hashToken(pair.RefreshToken)]; !ok {
		t.Fatal("expected new refresh token to be stored")
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
	if _, ok := refreshStore.tokens[currentHash]; ok {
		t.Fatal("expected refresh token to be removed")
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
