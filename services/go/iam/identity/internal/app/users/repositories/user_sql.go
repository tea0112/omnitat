package repositories

import (
	"fmt"
	"strings"

	"github.com/tea0112/omnitat/services/go/iam/identity/internal/app/users/models"
)

func BuildCreateUserSql() string {
	f := models.UserFields()

	columns := []string{
		f.IdColumn,
		f.EmailColumn,
		f.PasswordHashColumn,
		f.FirstNameColumn,
		f.LastNameColumn,
		f.IsActiveColumn,
		f.CreatedAtColumn,
		f.UpdatedAtColumn,
	}

	placeholders := make([]string, len(columns))
	for i := range columns {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
	}

	sql := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		f.TableName,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
	)

	return sql
}

func BuildFindUserByEmailSql() string {
	f := models.UserFields()

	return fmt.Sprintf(
		"SELECT %s, %s, %s, %s, %s, %s, %s, %s, %s FROM %s WHERE LOWER(%s) = LOWER($1) AND %s IS NULL LIMIT 1",
		f.IdColumn,
		f.EmailColumn,
		f.PasswordHashColumn,
		f.FirstNameColumn,
		f.LastNameColumn,
		f.IsActiveColumn,
		f.CreatedAtColumn,
		f.UpdatedAtColumn,
		f.DeletedAtColumn,
		f.TableName,
		f.EmailColumn,
		f.DeletedAtColumn,
	)
}

func BuildFindUserByIDSql() string {
	f := models.UserFields()

	return fmt.Sprintf(
		"SELECT %s, %s, %s, %s, %s, %s, %s, %s, %s FROM %s WHERE %s = $1 AND %s IS NULL LIMIT 1",
		f.IdColumn,
		f.EmailColumn,
		f.PasswordHashColumn,
		f.FirstNameColumn,
		f.LastNameColumn,
		f.IsActiveColumn,
		f.CreatedAtColumn,
		f.UpdatedAtColumn,
		f.DeletedAtColumn,
		f.TableName,
		f.IdColumn,
		f.DeletedAtColumn,
	)
}
