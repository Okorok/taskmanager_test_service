package infrastructure

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"
)

type txKey struct{}

func WithTx(ctx context.Context, tx *sqlx.Tx) context.Context {
	return context.WithValue(ctx, txKey{}, tx)
}

func TxFromCtx(ctx context.Context) (*sqlx.Tx, bool) {
	tx, ok := ctx.Value(txKey{}).(*sqlx.Tx)
	return tx, ok
}

type UnitOfWork struct {
	db *sqlx.DB
}

func NewUnitOfWork(db *sqlx.DB) *UnitOfWork {
	return &UnitOfWork{db: db}
}

func (t *UnitOfWork) Do(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := t.db.BeginTxx(ctx, &sql.TxOptions{})
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	ctxWithTx := WithTx(ctx, tx)
	if err := fn(ctxWithTx); err != nil {
		return err
	}

	return tx.Commit()
}
