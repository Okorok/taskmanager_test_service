package repository

import (
	"context"
	"database/sql"

	"taskmanager/internal/entity"
	"taskmanager/internal/infrastructure"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

type UserRepository struct {
	db *sqlx.DB
}

func NewUserRepository(db *sqlx.DB) *UserRepository {
	return &UserRepository{db: db}
}

type CreateUserRequest struct {
	Email        string
	PasswordHash string
	Name         string
}

const queryInsertUser = `
	INSERT INTO users (email, password_hash, name)
	VALUES (?, ?, ?)
`

func (r *UserRepository) Create(ctx context.Context, request CreateUserRequest) (*entity.User, error) {
	res, err := queryExecutor(ctx, r.db).ExecContext(ctx, queryInsertUser,
		request.Email,
		request.PasswordHash,
		request.Name,
	)
	if err != nil {
		if infrastructure.IsDuplicateKey(err) {
			return nil, infrastructure.ErrAlreadyExists
		}

		return nil, errors.WithStack(errors.Wrap(err, "failed to insert user"))
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, errors.WithStack(errors.Wrap(err, "failed to get inserted user id"))
	}

	return r.GetByID(ctx, id)
}

const queryGetUserByID = `
	SELECT id, email, password_hash, name, created_at
	FROM users
	WHERE id = ?
`

func (r *UserRepository) GetByID(ctx context.Context, id int64) (*entity.User, error) {
	var user entity.User
	if err := sqlx.GetContext(ctx, queryExecutor(ctx, r.db), &user, queryGetUserByID, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, infrastructure.ErrNotFound
		}

		return nil, errors.WithStack(errors.Wrap(err, "failed to get user by id"))
	}

	return &user, nil
}

const queryGetUserByEmail = `
	SELECT id, email, password_hash, name, created_at
	FROM users
	WHERE email = ?
`

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*entity.User, error) {
	var user entity.User
	if err := sqlx.GetContext(ctx, queryExecutor(ctx, r.db), &user, queryGetUserByEmail, email); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, infrastructure.ErrNotFound
		}

		return nil, errors.WithStack(errors.Wrap(err, "failed to get user by email"))
	}

	return &user, nil
}
