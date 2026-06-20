package service

import (
	"context"

	"taskmanager/internal/entity"
)

type UnitOfWork interface {
	Do(ctx context.Context, fn func(ctx context.Context) error) error
}

type MembershipRepository interface {
	Get(ctx context.Context, teamID, userID int64) (*entity.TeamMember, error)
}
