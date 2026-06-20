//go:build integration

package testdb

import (
	"context"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/mysql"
)

var (
	once   sync.Once
	dsn    string
	dsnErr error
)

// New поднимает (один раз на пакет) контейнер MySQL с применённой миграцией и
// возвращает свежий пул соединений к нему. Контейнер переиспользуется между
// тестами пакета, а очистку выполняет Ryuk при завершении тест-сессии.
func New(t *testing.T) *sqlx.DB {
	t.Helper()

	once.Do(startContainer)
	require.NoError(t, dsnErr, "failed to start mysql container")

	db, err := sqlx.Connect("mysql", dsn)
	require.NoError(t, err)

	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(20)
	db.SetConnMaxLifetime(time.Minute)

	return db
}

func startContainer() {
	ctx := context.Background()

	container, err := mysql.Run(ctx, "mysql:8.0",
		mysql.WithDatabase("app"),
		mysql.WithUsername("app"),
		mysql.WithPassword("app"),
		mysql.WithScripts(migrationPath()),
	)
	if err != nil {
		dsnErr = err
		return
	}

	dsn, dsnErr = container.ConnectionString(ctx, "parseTime=true", "multiStatements=true")
}

func migrationPath() string {
	_, thisFile, _, _ := runtime.Caller(0)

	return filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "migrations", "001_init_schema.up.sql")
}
