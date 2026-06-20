package entity

import "time"

type Team struct {
	ID        int64     `db:"id"`
	Name      string    `db:"name"`
	CreatedBy int64     `db:"created_by"`
	CreatedAt time.Time `db:"created_at"`
}
