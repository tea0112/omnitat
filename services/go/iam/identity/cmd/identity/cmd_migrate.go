package main

import (
	"context"

	libDatabase "github.com/tea0112/omnitat/libs/go/database"
	"github.com/tea0112/omnitat/services/go/iam/identity/internal/config"
)

func runMigration(cfg *config.Config) error {
	db, err := libDatabase.NewDatabaseConnection(&cfg.Database)
	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.ExecContext(context.Background(), `
		CREATE TABLE IF NOT EXISTS refresh_tokens (
			id UUID PRIMARY KEY,
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			token_hash VARCHAR(64) NOT NULL UNIQUE,
			expires_at TIMESTAMPTZ NOT NULL,
			revoked_at TIMESTAMPTZ NULL,
			last_used_at TIMESTAMPTZ NULL,
			user_agent TEXT NULL,
			ip_address VARCHAR(64) NULL,
			created_at TIMESTAMPTZ NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL
		);

		CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_id
			ON refresh_tokens(user_id);

		CREATE INDEX IF NOT EXISTS idx_refresh_tokens_expires_at
			ON refresh_tokens(expires_at);
	`)

	return err
}
