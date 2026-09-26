package example

import "time"

type Example struct {
	CreatedAt time.Time `db:"created_at"`
	ID        string    `db:"id"`
	Name      string    `db:"name"`
}
