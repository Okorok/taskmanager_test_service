package team_handler

import (
	"time"

	"taskmanager/internal/entity"
)

type teamResponse struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	CreatedBy int64     `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}

func toTeamResponse(team *entity.Team) teamResponse {
	return teamResponse{
		ID:        team.ID,
		Name:      team.Name,
		CreatedBy: team.CreatedBy,
		CreatedAt: team.CreatedAt,
	}
}

func toTeamResponses(teams []entity.Team) []teamResponse {
	result := make([]teamResponse, 0, len(teams))
	for i := range teams {
		result = append(result, toTeamResponse(&teams[i]))
	}

	return result
}
