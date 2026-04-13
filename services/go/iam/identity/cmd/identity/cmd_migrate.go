package main

import (
	"context"
	"database/sql"
	"fmt"

	libDatabase "github.com/tea0112/omnitat/libs/go/database"
	authRepositories "github.com/tea0112/omnitat/services/go/iam/identity/internal/app/auth/repositories"
	"github.com/tea0112/omnitat/services/go/iam/identity/internal/config"
)

func runMigration(cfg *config.Config) error {
	db, err := libDatabase.NewDatabaseConnection(&cfg.Database)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	ctx := context.Background()
	if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		name TEXT PRIMARY KEY,
		applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	)`); err != nil {
		return fmt.Errorf("create schema_migrations table: %w", err)
	}

	migrations := []struct {
		name string
		sql  string
	}{
		{name: "001_create_refresh_tokens_table", sql: authRepositories.BuildCreateRefreshTokensTableSQL()},
	}

	for i, statement := range authRepositories.BuildCreateRefreshTokensIndexesSQL() {
		migrations = append(migrations, struct {
			name string
			sql  string
		}{
			name: fmt.Sprintf("002_refresh_tokens_index_%d", i+1),
			sql:  statement,
		})
	}

	for _, migration := range migrations {
		if err := applyMigration(ctx, db, migration.name, migration.sql); err != nil {
			return err
		}
	}

	return nil
}

func applyMigration(ctx context.Context, db *sql.DB, name string, statement string) error {
	var applied bool
	err := db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE name = $1)", name).Scan(&applied)
	if err != nil {
		return fmt.Errorf("check migration %s: %w", name, err)
	}
	if applied {
		return nil
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin migration %s: %w", name, err)
	}

	if _, err := tx.ExecContext(ctx, statement); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("run migration %s: %w", name, err)
	}

	if _, err := tx.ExecContext(ctx, "INSERT INTO schema_migrations (name) VALUES ($1)", name); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("record migration %s: %w", name, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit migration %s: %w", name, err)
	}

	return nil
}
