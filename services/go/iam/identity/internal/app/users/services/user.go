package services

import (
	"context"
	"log/slog"
	"strings"

	"github.com/tea0112/omnitat/libs/go/datetime"
	"github.com/tea0112/omnitat/libs/go/security"
	"github.com/tea0112/omnitat/services/go/iam/identity/internal/app/users/models"
	"github.com/tea0112/omnitat/services/go/iam/identity/internal/app/users/repositories"
	"github.com/tea0112/omnitat/services/go/iam/identity/internal/app/users/transport/http/dto"
	"github.com/tea0112/omnitat/services/go/iam/identity/pkg/apperrors"
)

type UserServiceImpl struct {
	clock          datetime.Clock
	userRepository UserRepository
}

func NewUserService(
	userRepo *repositories.UserRepositoryImpl,
	realClock datetime.Clock,
) *UserServiceImpl {
	return &UserServiceImpl{
		userRepository: userRepo,
		clock:          realClock,
	}
}

func (svc *UserServiceImpl) CreateUser(ctx context.Context, createUserDTO *dto.CreateUserDTO) (*models.User, error) {
	now := svc.clock.Now()
	user, err := models.NewUser(now)
	if err != nil {
		slog.Error(err.Error())
		return nil, err
	}

	user.Email = &createUserDTO.Email
	user.FirstName = &createUserDTO.FirstName
	user.LastName = &createUserDTO.LastName

	passwordHash, err := security.HashPassword(createUserDTO.Password)
	if err != nil {
		return nil, err
	}
	user.PasswordHash = &passwordHash

	err = svc.userRepository.CreateUser(ctx, user)
	if err != nil {
		slog.Error("failed to create user: " + err.Error())
		if isDuplicateKeyError(err) {
			return nil, apperrors.ErrEmailAlreadyExists
		}
		return nil, err
	}

	slog.Info("user created", "user_id", user.Id.String(), "email", createUserDTO.Email)

	return user, nil
}

func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique constraint")
}
