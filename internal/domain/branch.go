package domain

type Branch struct {
	ID   int    `gorm:"primaryKey" db:"id" json:"id"`
	Name string `gorm:"unique;not null" db:"name" json:"name"`
}
