package services

import (
	"context"
	"log/slog"

	"github.com/tea0112/omnitat/libs/go/datetime"
	"github.com/tea0112/omnitat/libs/go/security"
	"github.com/tea0112/omnitat/services/go/iam/identity/internal/app/users/models"
	"github.com/tea0112/omnitat/services/go/iam/identity/internal/app/users/repositories"
	"github.com/tea0112/omnitat/services/go/iam/identity/internal/app/users/transport/http/dto"
)

// TODO: business logic, interface repo, domain rule
// TODO: Unit of Work

type UserServiceImpl struct {
	clock          datetime.Clock
	userRepository UserRepository
}

func NewUserService(
	userRepositoryImpl *repositories.UserRepositoryImpl,
	realClock datetime.Clock,
) *UserServiceImpl {
	return &UserServiceImpl{
		userRepository: userRepositoryImpl,
		clock:          realClock,
	}
}

func (svc *UserServiceImpl) CreateUser(ctx context.Context, createUserDTO *dto.CreateUserDTO) (*models.User, error) {
	user, err := models.NewUser(svc.clock.Now())
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
		return nil, err
	}

	return user, nil
}
