package domain

type Currency struct {
	ID          int    `gorm:"primaryKey" db:"id" json:"id"`
	Code        string `gorm:"type:char(3);not null" db:"code" json:"code"`
	NumericCode int    `gorm:"not null" db:"numeric_code" json:"numeric_code"`
	NameRU      string `gorm:"unique;not null" db:"name_ru" json:"name_ru"`
}
