//go:build integration

package repository_test

import (
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

var userCounter atomic.Int64

// createUser вставляет пользователя с уникальным email и возвращает его id.
func createUser(t *testing.T, db *sqlx.DB, name string) int64 {
	t.Helper()

	email := fmt.Sprintf("user-%d@test.local", userCounter.Add(1))

	res, err := db.ExecContext(t.Context(),
		"INSERT INTO users (email, password_hash, name) VALUES (?, ?, ?)",
		email, "hash", name,
	)
	require.NoError(t, err)

	id, err := res.LastInsertId()
	require.NoError(t, err)

	return id
}

// createTeam вставляет команду и возвращает её id.
func createTeam(t *testing.T, db *sqlx.DB, createdBy int64, name string) int64 {
	t.Helper()

	res, err := db.ExecContext(t.Context(),
		"INSERT INTO teams (name, created_by) VALUES (?, ?)",
		name, createdBy,
	)
	require.NoError(t, err)

	id, err := res.LastInsertId()
	require.NoError(t, err)

	return id
}

// addMember добавляет пользователя в команду с заданной ролью.
func addMember(t *testing.T, db *sqlx.DB, teamID, userID int64, role string) {
	t.Helper()

	_, err := db.ExecContext(t.Context(),
		"INSERT INTO team_members (team_id, user_id, role) VALUES (?, ?, ?)",
		teamID, userID, role,
	)
	require.NoError(t, err)
}

// insertTask вставляет задачу напрямую (минуя сервис) — удобно для подготовки данных,
// в том числе для заведомо некорректных (assignee вне команды).
func insertTask(t *testing.T, db *sqlx.DB, teamID, createdBy int64, status string, assignee *int64) int64 {
	t.Helper()

	res, err := db.ExecContext(t.Context(),
		"INSERT INTO tasks (team_id, title, description, status, priority, assignee_id, created_by) VALUES (?, ?, ?, ?, ?, ?, ?)",
		teamID, "title", "description", status, "medium", assignee, createdBy,
	)
	require.NoError(t, err)

	id, err := res.LastInsertId()
	require.NoError(t, err)

	return id
}
