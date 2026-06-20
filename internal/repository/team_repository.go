package repository

import (
	"context"
	"database/sql"

	"taskmanager/internal/entity"
	"taskmanager/internal/infrastructure"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

type TeamRepository struct {
	db *sqlx.DB
}

func NewTeamRepository(db *sqlx.DB) *TeamRepository {
	return &TeamRepository{db: db}
}

type CreateTeamRequest struct {
	Name      string
	CreatedBy int64
}

const queryInsertTeam = `
	INSERT INTO teams (name, created_by)
	VALUES (?, ?)
`

func (r *TeamRepository) Create(ctx context.Context, request CreateTeamRequest) (*entity.Team, error) {
	res, err := queryExecutor(ctx, r.db).ExecContext(ctx, queryInsertTeam, request.Name, request.CreatedBy)
	if err != nil {
		return nil, errors.WithStack(errors.Wrap(err, "failed to insert team"))
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, errors.WithStack(errors.Wrap(err, "failed to get inserted team id"))
	}

	return r.GetByID(ctx, id)
}

const queryGetTeamByID = `
	SELECT id, name, created_by, created_at
	FROM teams
	WHERE id = ?
`

func (r *TeamRepository) GetByID(ctx context.Context, id int64) (*entity.Team, error) {
	var team entity.Team
	if err := sqlx.GetContext(ctx, queryExecutor(ctx, r.db), &team, queryGetTeamByID, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, infrastructure.ErrNotFound
		}

		return nil, errors.WithStack(errors.Wrap(err, "failed to get team by id"))
	}

	return &team, nil
}

const queryListTeamsByUser = `
	SELECT t.id, t.name, t.created_by, t.created_at
	FROM teams t
	JOIN team_members tm ON tm.team_id = t.id
	WHERE tm.user_id = ?
	ORDER BY t.id
`

func (r *TeamRepository) ListByUser(ctx context.Context, userID int64) ([]entity.Team, error) {
	var teams []entity.Team
	if err := sqlx.SelectContext(ctx, queryExecutor(ctx, r.db), &teams, queryListTeamsByUser, userID); err != nil {
		return nil, errors.WithStack(errors.Wrap(err, "failed to list teams by user"))
	}

	return teams, nil
}
