package domain

type Currency struct {
	ID          int    `db:"id"`
	Code        string `db:"code"`
	NumericCode int    `db:"numeric_code"`
	NameRU      string `db:"name_ru"`
}
