package entity

import "time"

const (
	TaskFieldCreated     = "created"
	TaskFieldTitle       = "title"
	TaskFieldDescription = "description"
	TaskFieldStatus      = "status"
	TaskFieldPriority    = "priority"
	TaskFieldAssignee    = "assignee_id"
)

type TaskHistory struct {
	ID        int64     `db:"id"`
	TaskID    int64     `db:"task_id"`
	ChangedBy int64     `db:"changed_by"`
	Field     string    `db:"field"`
	OldValue  *string   `db:"old_value"`
	NewValue  *string   `db:"new_value"`
	ChangedAt time.Time `db:"changed_at"`
}
