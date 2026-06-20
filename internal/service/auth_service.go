package service

import (
	"context"

	"taskmanager/internal/entity"
	"taskmanager/internal/infrastructure"
	"taskmanager/internal/repository"

	"github.com/pkg/errors"
)

var (
	ErrEmailAlreadyTaken  = errors.New("email already taken")
	ErrInvalidCredentials = errors.New("invalid credentials")
)

type UserRepository interface {
	Create(ctx context.Context, request repository.CreateUserRequest) (*entity.User, error)
	GetByEmail(ctx context.Context, email string) (*entity.User, error)
	GetByID(ctx context.Context, id int64) (*entity.User, error)
}

type PasswordHasher interface {
	Hash(password string) (string, error)
	Check(hash, password string) bool
}

type TokenIssuer interface {
	Generate(userID int64) (string, error)
}

type AuthService struct {
	users  UserRepository
	hasher PasswordHasher
	tokens TokenIssuer
}

func NewAuthService(users UserRepository, hasher PasswordHasher, tokens TokenIssuer) *AuthService {
	return &AuthService{
		users:  users,
		hasher: hasher,
		tokens: tokens,
	}
}

type RegisterCommand struct {
	Email    string
	Password string
	Name     string
}

func (s *AuthService) Register(ctx context.Context, cmd RegisterCommand) (*entity.User, error) {
	hash, err := s.hasher.Hash(cmd.Password)
	if err != nil {
		return nil, err
	}

	user, err := s.users.Create(ctx, repository.CreateUserRequest{
		Email:        cmd.Email,
		PasswordHash: hash,
		Name:         cmd.Name,
	})
	if err != nil {
		if errors.Is(err, infrastructure.ErrAlreadyExists) {
			return nil, ErrEmailAlreadyTaken
		}

		return nil, err
	}

	return user, nil
}

type LoginCommand struct {
	Email    string
	Password string
}

func (s *AuthService) Login(ctx context.Context, cmd LoginCommand) (string, error) {
	user, err := s.users.GetByEmail(ctx, cmd.Email)
	if err != nil {
		if errors.Is(err, infrastructure.ErrNotFound) {
			return "", ErrInvalidCredentials
		}

		return "", err
	}

	if !s.hasher.Check(user.PasswordHash, cmd.Password) {
		return "", ErrInvalidCredentials
	}

	token, err := s.tokens.Generate(user.ID)
	if err != nil {
		return "", err
	}

	return token, nil
}
