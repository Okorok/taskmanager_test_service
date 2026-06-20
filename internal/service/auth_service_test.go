package service

import (
	"context"
	"errors"
	"testing"

	"taskmanager/internal/entity"
	"taskmanager/internal/infrastructure"
	"taskmanager/internal/repository"
	"taskmanager/internal/service/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func Test_register_successfully(t *testing.T) {
	created := &entity.User{ID: 1, Email: "user@example.com", Name: "User"}

	users := mocks.NewMockUserRepository(t)
	users.EXPECT().
		Create(mock.Anything, mock.Anything).
		Run(func(_ context.Context, req repository.CreateUserRequest) {
			assert.Equal(t, "hashed", req.PasswordHash)
		}).
		Return(created, nil)

	hasher := mocks.NewMockPasswordHasher(t)
	hasher.EXPECT().Hash("secret123").Return("hashed", nil)

	svc := NewAuthService(users, hasher, mocks.NewMockTokenIssuer(t))

	user, err := svc.Register(t.Context(), RegisterCommand{Email: "user@example.com", Password: "secret123", Name: "User"})

	require.NoError(t, err)
	assert.Equal(t, created, user)
}

func Test_register_email_already_taken(t *testing.T) {
	users := mocks.NewMockUserRepository(t)
	users.EXPECT().Create(mock.Anything, mock.Anything).Return(nil, infrastructure.ErrAlreadyExists)

	hasher := mocks.NewMockPasswordHasher(t)
	hasher.EXPECT().Hash(mock.Anything).Return("hashed", nil)

	svc := NewAuthService(users, hasher, mocks.NewMockTokenIssuer(t))

	user, err := svc.Register(t.Context(), RegisterCommand{Email: "user@example.com", Password: "secret123", Name: "User"})

	assert.ErrorIs(t, err, ErrEmailAlreadyTaken)
	assert.Nil(t, user)
}

func Test_register_hash_error(t *testing.T) {
	hasher := mocks.NewMockPasswordHasher(t)
	hasher.EXPECT().Hash(mock.Anything).Return("", errors.New("hash error"))

	svc := NewAuthService(mocks.NewMockUserRepository(t), hasher, mocks.NewMockTokenIssuer(t))

	user, err := svc.Register(t.Context(), RegisterCommand{Email: "user@example.com", Password: "secret123", Name: "User"})

	assert.Error(t, err)
	assert.Nil(t, user)
}

func Test_login_successfully(t *testing.T) {
	stored := &entity.User{ID: 7, Email: "user@example.com", PasswordHash: "hashed"}

	users := mocks.NewMockUserRepository(t)
	users.EXPECT().GetByEmail(mock.Anything, "user@example.com").Return(stored, nil)

	hasher := mocks.NewMockPasswordHasher(t)
	hasher.EXPECT().Check("hashed", "secret123").Return(true)

	tokens := mocks.NewMockTokenIssuer(t)
	tokens.EXPECT().Generate(int64(7)).Return("signed-token", nil)

	svc := NewAuthService(users, hasher, tokens)

	token, err := svc.Login(t.Context(), LoginCommand{Email: "user@example.com", Password: "secret123"})

	require.NoError(t, err)
	assert.Equal(t, "signed-token", token)
}

func Test_login_user_not_found(t *testing.T) {
	users := mocks.NewMockUserRepository(t)
	users.EXPECT().GetByEmail(mock.Anything, mock.Anything).Return(nil, infrastructure.ErrNotFound)

	svc := NewAuthService(users, mocks.NewMockPasswordHasher(t), mocks.NewMockTokenIssuer(t))

	token, err := svc.Login(t.Context(), LoginCommand{Email: "missing@example.com", Password: "secret123"})

	assert.ErrorIs(t, err, ErrInvalidCredentials)
	assert.Empty(t, token)
}

func Test_login_wrong_password(t *testing.T) {
	stored := &entity.User{ID: 7, PasswordHash: "hashed"}

	users := mocks.NewMockUserRepository(t)
	users.EXPECT().GetByEmail(mock.Anything, mock.Anything).Return(stored, nil)

	hasher := mocks.NewMockPasswordHasher(t)
	hasher.EXPECT().Check(mock.Anything, mock.Anything).Return(false)

	svc := NewAuthService(users, hasher, mocks.NewMockTokenIssuer(t))

	token, err := svc.Login(t.Context(), LoginCommand{Email: "user@example.com", Password: "wrong"})

	assert.ErrorIs(t, err, ErrInvalidCredentials)
	assert.Empty(t, token)
}

func Test_login_unexpected_repo_error(t *testing.T) {
	users := mocks.NewMockUserRepository(t)
	users.EXPECT().GetByEmail(mock.Anything, mock.Anything).Return(nil, errors.New("db error"))

	svc := NewAuthService(users, mocks.NewMockPasswordHasher(t), mocks.NewMockTokenIssuer(t))

	token, err := svc.Login(t.Context(), LoginCommand{Email: "user@example.com", Password: "secret123"})

	assert.Error(t, err)
	assert.NotErrorIs(t, err, ErrInvalidCredentials)
	assert.Empty(t, token)
}
