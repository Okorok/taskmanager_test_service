package repository

import (
	"context"
	"database/sql"

	"taskmanager/internal/entity"
	"taskmanager/internal/infrastructure"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

type TeamMemberRepository struct {
	db *sqlx.DB
}

func NewTeamMemberRepository(db *sqlx.DB) *TeamMemberRepository {
	return &TeamMemberRepository{db: db}
}

type AddTeamMemberRequest struct {
	TeamID int64
	UserID int64
	Role   entity.TeamRole
}

const queryInsertTeamMember = `
	INSERT INTO team_members (team_id, user_id, role)
	VALUES (?, ?, ?)
`

func (r *TeamMemberRepository) Add(ctx context.Context, request AddTeamMemberRequest) error {
	_, err := queryExecutor(ctx, r.db).ExecContext(ctx, queryInsertTeamMember,
		request.TeamID,
		request.UserID,
		request.Role,
	)
	if err != nil {
		if infrastructure.IsDuplicateKey(err) {
			return infrastructure.ErrAlreadyExists
		}

		return errors.WithStack(errors.Wrap(err, "failed to insert team member"))
	}

	return nil
}

const queryGetTeamMember = `
	SELECT team_id, user_id, role, joined_at
	FROM team_members
	WHERE team_id = ? AND user_id = ?
`

func (r *TeamMemberRepository) Get(ctx context.Context, teamID, userID int64) (*entity.TeamMember, error) {
	var member entity.TeamMember
	if err := sqlx.GetContext(ctx, queryExecutor(ctx, r.db), &member, queryGetTeamMember, teamID, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, infrastructure.ErrNotFound
		}

		return nil, errors.WithStack(errors.Wrap(err, "failed to get team member"))
	}

	return &member, nil
}
