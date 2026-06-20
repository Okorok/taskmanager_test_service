//go:build integration

package repository_test

import (
	"testing"

	"taskmanager/internal/infrastructure"
	"taskmanager/internal/infrastructure/testdb"
	"taskmanager/internal/repository"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_integration_create_and_get_user(t *testing.T) {
	db := testdb.New(t)
	defer db.Close()

	repo := repository.NewUserRepository(db)

	created, err := repo.Create(t.Context(), repository.CreateUserRequest{
		Email:        "create-user@test.local",
		PasswordHash: "hash",
		Name:         "User",
	})
	require.NoError(t, err)
	assert.NotZero(t, created.ID)

	byID, err := repo.GetByID(t.Context(), created.ID)
	require.NoError(t, err)
	assert.Equal(t, created, byID)

	byEmail, err := repo.GetByEmail(t.Context(), "create-user@test.local")
	require.NoError(t, err)
	assert.Equal(t, created.ID, byEmail.ID)
}

func Test_integration_create_user_duplicate_email(t *testing.T) {
	db := testdb.New(t)
	defer db.Close()

	repo := repository.NewUserRepository(db)

	_, err := repo.Create(t.Context(), repository.CreateUserRequest{
		Email:        "dup@test.local",
		PasswordHash: "hash",
		Name:         "User",
	})
	require.NoError(t, err)

	_, err = repo.Create(t.Context(), repository.CreateUserRequest{
		Email:        "dup@test.local",
		PasswordHash: "hash",
		Name:         "User2",
	})

	assert.ErrorIs(t, err, infrastructure.ErrAlreadyExists)
}

func Test_integration_get_user_not_found(t *testing.T) {
	db := testdb.New(t)
	defer db.Close()

	repo := repository.NewUserRepository(db)

	_, err := repo.GetByEmail(t.Context(), "missing@test.local")

	assert.ErrorIs(t, err, infrastructure.ErrNotFound)
}
