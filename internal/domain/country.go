package domain

type Country struct {
	ID          int    `gorm:"primaryKey" db:"id" json:"id"`
	CountryCode string `gorm:"column:country_code" db:"country_code" json:"country_code"`
	NameRU      string `gorm:"unique;not null;column:name_ru" db:"name_ru" json:"name_ru"`
	NumericCode *int   `gorm:"unique;column:numeric_code" json:"numeric_code"`
}
