package entity

import "time"

type TeamRole string

const (
	TeamRoleOwner  TeamRole = "owner"
	TeamRoleAdmin  TeamRole = "admin"
	TeamRoleMember TeamRole = "member"
)

func (r TeamRole) CanInvite() bool {
	return r == TeamRoleOwner || r == TeamRoleAdmin
}

func (r TeamRole) IsValid() bool {
	switch r {
	case TeamRoleOwner, TeamRoleAdmin, TeamRoleMember:
		return true
	default:
		return false
	}
}

type TeamMember struct {
	TeamID   int64     `db:"team_id"`
	UserID   int64     `db:"user_id"`
	Role     TeamRole  `db:"role"`
	JoinedAt time.Time `db:"joined_at"`
}
