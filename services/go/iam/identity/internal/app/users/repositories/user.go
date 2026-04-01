package repositories

import (
	"context"
	"database/sql"

	"github.com/tea0112/omnitat/services/go/iam/identity/internal/app/users/models"
)

type UserRepositoryImpl struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepositoryImpl {
	return &UserRepositoryImpl{db: db}
}

func (i *UserRepositoryImpl) CreateUser(ctx context.Context, user *models.User) error {
	sql := BuildCreateUserSql()

	_, err := i.db.ExecContext(ctx, sql,
		user.Id,
		user.Email,
		user.PasswordHash,
		user.FirstName,
		user.LastName,
		user.IsActive,
		user.CreatedAt,
		user.UpdatedAt,
	)
	if err != nil {
		return err
	}

	return nil
}
