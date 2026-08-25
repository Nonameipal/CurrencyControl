package domain

type Branch struct {
	ID   int    `db:"id"`
	Name string `db:"name"`
}
