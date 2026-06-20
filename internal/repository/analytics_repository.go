package repository

import (
	"context"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

type AnalyticsRepository struct {
	db *sqlx.DB
}

func NewAnalyticsRepository(db *sqlx.DB) *AnalyticsRepository {
	return &AnalyticsRepository{db: db}
}

type TeamStats struct {
	TeamID             int64  `db:"team_id"`
	TeamName           string `db:"team_name"`
	MembersCount       int64  `db:"members_count"`
	DoneTasksLast7Days int64  `db:"done_tasks_last_7_days"`
}

const queryTeamStats = `
	SELECT
	    t.id   AS team_id,
	    t.name AS team_name,
	    COUNT(DISTINCT tm.user_id) AS members_count,
	    COUNT(DISTINCT CASE
	        WHEN tk.status = 'done' AND tk.updated_at >= (NOW() - INTERVAL 7 DAY)
	        THEN tk.id
	    END) AS done_tasks_last_7_days
	FROM teams t
	LEFT JOIN team_members tm ON tm.team_id = t.id
	LEFT JOIN tasks tk        ON tk.team_id = t.id
	GROUP BY t.id, t.name
	ORDER BY t.id
`

func (r *AnalyticsRepository) TeamStats(ctx context.Context) ([]TeamStats, error) {
	var stats []TeamStats
	if err := sqlx.SelectContext(ctx, queryExecutor(ctx, r.db), &stats, queryTeamStats); err != nil {
		return nil, errors.WithStack(errors.Wrap(err, "failed to get team stats"))
	}

	return stats, nil
}

type TopCreator struct {
	TeamID       int64 `db:"team_id"`
	UserID       int64 `db:"user_id"`
	TasksCreated int64 `db:"tasks_created"`
	Rank         int64 `db:"rnk"`
}

const queryTopCreators = `
	SELECT team_id, user_id, tasks_created, rnk
	FROM (
	    SELECT
	        tk.team_id,
	        tk.created_by AS user_id,
	        COUNT(*)      AS tasks_created,
	        ROW_NUMBER() OVER (
	            PARTITION BY tk.team_id
	            ORDER BY COUNT(*) DESC, tk.created_by
	        ) AS rnk
	    FROM tasks tk
	    WHERE tk.created_at >= (NOW() - INTERVAL 1 MONTH)
	    GROUP BY tk.team_id, tk.created_by
	) ranked
	WHERE rnk <= 3
	ORDER BY team_id, rnk
`

func (r *AnalyticsRepository) TopCreatorsPerTeam(ctx context.Context) ([]TopCreator, error) {
	var creators []TopCreator
	if err := sqlx.SelectContext(ctx, queryExecutor(ctx, r.db), &creators, queryTopCreators); err != nil {
		return nil, errors.WithStack(errors.Wrap(err, "failed to get top creators"))
	}

	return creators, nil
}

type InconsistentTask struct {
	TaskID     int64 `db:"task_id"`
	TeamID     int64 `db:"team_id"`
	AssigneeID int64 `db:"assignee_id"`
}

const queryInconsistentTasks = `
	SELECT tk.id AS task_id, tk.team_id, tk.assignee_id
	FROM tasks tk
	WHERE tk.assignee_id IS NOT NULL
	  AND NOT EXISTS (
	      SELECT 1
	      FROM team_members tm
	      WHERE tm.team_id = tk.team_id
	        AND tm.user_id = tk.assignee_id
	  )
	ORDER BY tk.id
`

func (r *AnalyticsRepository) InconsistentTasks(ctx context.Context) ([]InconsistentTask, error) {
	var tasks []InconsistentTask
	if err := sqlx.SelectContext(ctx, queryExecutor(ctx, r.db), &tasks, queryInconsistentTasks); err != nil {
		return nil, errors.WithStack(errors.Wrap(err, "failed to get inconsistent tasks"))
	}

	return tasks, nil
}
