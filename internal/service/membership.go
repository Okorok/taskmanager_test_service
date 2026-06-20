package service

import (
	"context"

	"taskmanager/internal/infrastructure"

	"github.com/pkg/errors"
)

func isTeamMember(ctx context.Context, members MembershipRepository, teamID, userID int64) (bool, error) {
	_, err := members.Get(ctx, teamID, userID)
	if err != nil {
		if errors.Is(err, infrastructure.ErrNotFound) {
			return false, nil
		}

		return false, err
	}

	return true, nil
}
