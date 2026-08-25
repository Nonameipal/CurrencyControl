package domain

type Country struct {
	ID          int    `db:"id"`
	CountryCode string `db:"country_code"`
	NameRU      string `db:"name_ru"`
}
