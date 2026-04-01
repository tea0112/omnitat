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
