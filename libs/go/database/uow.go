package database

import (
	"context"
	"database/sql"
	"fmt"
)

type Transactor struct {
	db *sql.DB
}

func NewTransactor(db *sql.DB) *Transactor {
	return &Transactor{
		db: db,
	}
}

func (t *Transactor) WithTransaction(ctx context.Context, txFunc func(*sql.Tx) error, opts *sql.TxOptions) error {
	tx, err := t.db.BeginTx(ctx, opts)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	err = txFunc(tx)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("execute transaction: %w", err)
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}
