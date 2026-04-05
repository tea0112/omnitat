package repositories

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	libDatabase "github.com/tea0112/omnitat/libs/go/database"
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
	return err
}

func (i *UserRepositoryImpl) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	sqlQuery := BuildFindUserByEmailSql()

	return i.findOne(ctx, sqlQuery, email)
}

func (i *UserRepositoryImpl) FindByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	sqlQuery := BuildFindUserByIDSql()

	return i.findOne(ctx, sqlQuery, id)
}

func (i *UserRepositoryImpl) findOne(ctx context.Context, sqlQuery string, args ...any) (*models.User, error) {
	var user models.User
	var email sql.Null[string]
	var passwordHash sql.Null[string]
	var firstName sql.Null[string]
	var lastName sql.Null[string]
	var deletedAt sql.Null[time.Time]

	err := i.db.QueryRowContext(ctx, sqlQuery, args...).Scan(
		&user.Id,
		&email,
		&passwordHash,
		&firstName,
		&lastName,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
		&deletedAt,
	)
	if err != nil {
		return nil, err
	}

	user.Email = libDatabase.NullPtr(email)
	user.PasswordHash = libDatabase.NullPtr(passwordHash)
	user.FirstName = libDatabase.NullPtr(firstName)
	user.LastName = libDatabase.NullPtr(lastName)
	user.DeletedAt = libDatabase.NullPtr(deletedAt)

	return &user, nil
}
