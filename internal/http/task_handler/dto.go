package task_handler

import (
	"time"

	"taskmanager/internal/entity"
)

type taskResponse struct {
	ID          int64     `json:"id"`
	TeamID      int64     `json:"team_id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	Priority    string    `json:"priority"`
	AssigneeID  *int64    `json:"assignee_id"`
	CreatedBy   int64     `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func toTaskResponse(task *entity.Task) taskResponse {
	return taskResponse{
		ID:          task.ID,
		TeamID:      task.TeamID,
		Title:       task.Title,
		Description: task.Description,
		Status:      string(task.Status),
		Priority:    task.Priority,
		AssigneeID:  task.AssigneeID,
		CreatedBy:   task.CreatedBy,
		CreatedAt:   task.CreatedAt,
		UpdatedAt:   task.UpdatedAt,
	}
}

func toTaskResponses(tasks []entity.Task) []taskResponse {
	result := make([]taskResponse, 0, len(tasks))
	for i := range tasks {
		result = append(result, toTaskResponse(&tasks[i]))
	}

	return result
}

type historyResponse struct {
	ID        int64     `json:"id"`
	TaskID    int64     `json:"task_id"`
	ChangedBy int64     `json:"changed_by"`
	Field     string    `json:"field"`
	OldValue  *string   `json:"old_value"`
	NewValue  *string   `json:"new_value"`
	ChangedAt time.Time `json:"changed_at"`
}

func toHistoryResponses(history []entity.TaskHistory) []historyResponse {
	result := make([]historyResponse, 0, len(history))
	for i := range history {
		h := history[i]

		result = append(result, historyResponse{
			ID:        h.ID,
			TaskID:    h.TaskID,
			ChangedBy: h.ChangedBy,
			Field:     h.Field,
			OldValue:  h.OldValue,
			NewValue:  h.NewValue,
			ChangedAt: h.ChangedAt,
		})
	}

	return result
}

type commentResponse struct {
	ID        int64     `json:"id"`
	TaskID    int64     `json:"task_id"`
	UserID    int64     `json:"user_id"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}

func toCommentResponse(comment *entity.TaskComment) commentResponse {
	return commentResponse{
		ID:        comment.ID,
		TaskID:    comment.TaskID,
		UserID:    comment.UserID,
		Body:      comment.Body,
		CreatedAt: comment.CreatedAt,
	}
}

func toCommentResponses(comments []entity.TaskComment) []commentResponse {
	result := make([]commentResponse, 0, len(comments))
	for i := range comments {
		result = append(result, toCommentResponse(&comments[i]))
	}

	return result
}
