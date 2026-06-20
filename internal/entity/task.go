package entity

import "time"

type TaskStatus string

const (
	TaskStatusTodo       TaskStatus = "todo"
	TaskStatusInProgress TaskStatus = "in_progress"
	TaskStatusDone       TaskStatus = "done"
)

func (s TaskStatus) IsValid() bool {
	switch s {
	case TaskStatusTodo, TaskStatusInProgress, TaskStatusDone:
		return true
	default:
		return false
	}
}

type Task struct {
	ID          int64      `db:"id"`
	TeamID      int64      `db:"team_id"`
	Title       string     `db:"title"`
	Description string     `db:"description"`
	Status      TaskStatus `db:"status"`
	Priority    string     `db:"priority"`
	AssigneeID  *int64     `db:"assignee_id"`
	CreatedBy   int64      `db:"created_by"`
	CreatedAt   time.Time  `db:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at"`
}
