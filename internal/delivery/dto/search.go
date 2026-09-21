package dto

import "time"

type DashboardSearchRequest struct {
	CompanyName string  `json:"company_name"`
	Amount      float64 `json:"amount"`
	INN         string  `json:"inn"`
	BranchID    int     `json:"branch_id"`
}

type DashboardSearchResult struct {
	ID         int64     `json:"id"`
	Number     string    `json:"number"`
	Name       string    `json:"name"`        // Ф.И.О.
	LLC        string    `json:"llc"`         // Название компании/ЧДММ
	INN        string    `json:"inn"`         // ИНН
	ClientType string    `json:"client_type"` // Тип клиента (физ/юр лицо)
	Phones     []string  `json:"phones"`      // Телефон(ы)
	Accounts   []string  `json:"accounts"`    // Счета
	CreatedBy  string    `json:"created_by"`  // Кто создал
	CreatedAt  time.Time `json:"created_at"`  // Дата создания
	UpdatedAt  time.Time `json:"updated_at"`  // Дата изменения
}
