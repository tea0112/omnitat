package repositories

import (
	"fmt"
	"strings"

	"github.com/tea0112/omnitat/services/go/iam/identity/internal/app/auth/models"
)

func BuildCreateRefreshTokensTableSQL() string {
	f := models.RefreshTokenFields()

	return fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
		%s UUID PRIMARY KEY,
		%s UUID NOT NULL,
		%s UUID NOT NULL,
		%s TEXT NOT NULL,
		%s TIMESTAMPTZ NOT NULL,
		%s TIMESTAMPTZ NULL,
		%s TIMESTAMPTZ NULL,
		%s TEXT NULL,
		%s TEXT NULL,
		%s TIMESTAMPTZ NOT NULL,
		%s TIMESTAMPTZ NOT NULL
	)`,
		f.TableName,
		f.IDColumn,
		f.FamilyIDColumn,
		f.UserIDColumn,
		f.TokenHashColumn,
		f.ExpiresAtColumn,
		f.RevokedAtColumn,
		f.LastUsedAtColumn,
		f.UserAgentColumn,
		f.IPAddressColumn,
		f.CreatedAtColumn,
		f.UpdatedAtColumn,
	)
}

func BuildCreateRefreshTokensIndexesSQL() []string {
	f := models.RefreshTokenFields()

	return []string{
		fmt.Sprintf("CREATE UNIQUE INDEX IF NOT EXISTS idx_%s_token_hash ON %s (%s)", f.TableName, f.TableName, f.TokenHashColumn),
		fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%s_user_id ON %s (%s)", f.TableName, f.TableName, f.UserIDColumn),
		fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%s_family_id ON %s (%s)", f.TableName, f.TableName, f.FamilyIDColumn),
		fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%s_expires_at ON %s (%s)", f.TableName, f.TableName, f.ExpiresAtColumn),
		fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%s_user_id_revoked_at ON %s (%s, %s)", f.TableName, f.TableName, f.UserIDColumn, f.RevokedAtColumn),
		fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%s_family_id_revoked_at ON %s (%s, %s)", f.TableName, f.TableName, f.FamilyIDColumn, f.RevokedAtColumn),
	}
}

func BuildCreateRefreshTokenSQL() string {
	f := models.RefreshTokenFields()
	columns := []string{
		f.IDColumn,
		f.FamilyIDColumn,
		f.UserIDColumn,
		f.TokenHashColumn,
		f.ExpiresAtColumn,
		f.RevokedAtColumn,
		f.LastUsedAtColumn,
		f.UserAgentColumn,
		f.IPAddressColumn,
		f.CreatedAtColumn,
		f.UpdatedAtColumn,
	}

	placeholders := make([]string, len(columns))
	for i := range columns {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
	}

	return fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		f.TableName,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
	)
}

func BuildFindRefreshTokenByTokenHashSQL(forUpdate bool) string {
	f := models.RefreshTokenFields()
	query := fmt.Sprintf(
		"SELECT %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s FROM %s WHERE %s = $1 LIMIT 1",
		f.IDColumn,
		f.FamilyIDColumn,
		f.UserIDColumn,
		f.TokenHashColumn,
		f.ExpiresAtColumn,
		f.RevokedAtColumn,
		f.LastUsedAtColumn,
		f.UserAgentColumn,
		f.IPAddressColumn,
		f.CreatedAtColumn,
		f.UpdatedAtColumn,
		f.TableName,
		f.TokenHashColumn,
	)

	if forUpdate {
		query += " FOR UPDATE"
	}

	return query
}

func BuildRevokeRefreshTokenByTokenHashSQL() string {
	f := models.RefreshTokenFields()

	return fmt.Sprintf(
		"UPDATE %s SET %s = $2, %s = $2 WHERE %s = $1 AND %s IS NULL",
		f.TableName,
		f.RevokedAtColumn,
		f.UpdatedAtColumn,
		f.TokenHashColumn,
		f.RevokedAtColumn,
	)
}

func BuildRevokeRefreshTokenFamilySQL() string {
	f := models.RefreshTokenFields()

	return fmt.Sprintf(
		"UPDATE %s SET %s = $2, %s = $2 WHERE %s = $1 AND %s IS NULL",
		f.TableName,
		f.RevokedAtColumn,
		f.UpdatedAtColumn,
		f.FamilyIDColumn,
		f.RevokedAtColumn,
	)
}

func BuildRotateRefreshTokenCurrentSQL() string {
	f := models.RefreshTokenFields()

	return fmt.Sprintf(
		"UPDATE %s SET %s = $2, %s = $2, %s = $2 WHERE %s = $1",
		f.TableName,
		f.RevokedAtColumn,
		f.LastUsedAtColumn,
		f.UpdatedAtColumn,
		f.TokenHashColumn,
	)
}

func BuildListLatestSessionsByUserIDSQL() string {
	f := models.RefreshTokenFields()

	return fmt.Sprintf(
		"SELECT DISTINCT ON (%s) %s, %s, %s, %s, %s, %s, %s, %s, %s FROM %s WHERE %s = $1 AND %s > $2 ORDER BY %s, %s DESC",
		f.FamilyIDColumn,
		f.FamilyIDColumn,
		f.UserIDColumn,
		f.UserAgentColumn,
		f.IPAddressColumn,
		f.CreatedAtColumn,
		f.UpdatedAtColumn,
		f.LastUsedAtColumn,
		f.ExpiresAtColumn,
		f.RevokedAtColumn,
		f.TableName,
		f.UserIDColumn,
		f.ExpiresAtColumn,
		f.FamilyIDColumn,
		f.CreatedAtColumn,
	)
}
