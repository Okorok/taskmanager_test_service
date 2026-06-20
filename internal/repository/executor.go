package repository

import (
	"context"

	"taskmanager/internal/infrastructure"

	"github.com/jmoiron/sqlx"
)

// queryExecutor возвращает исполнитель запросов: транзакцию из контекста, если
// она там есть, иначе пул соединений. Так одни и те же методы репозитория
// работают и внутри UnitOfWork, и без транзакции.
func queryExecutor(ctx context.Context, db *sqlx.DB) sqlx.ExtContext {
	if tx, ok := infrastructure.TxFromCtx(ctx); ok {
		return tx
	}

	return db
}
