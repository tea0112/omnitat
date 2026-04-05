package database

import "database/sql"

func NullPtr[T any](value sql.Null[T]) *T {
	if !value.Valid {
		return nil
	}

	return &value.V
}
