package dto

type DashboardSearchRequest struct {
	CompanyName string  `json:"company_name"`
	Amount      float64 `json:"amount"`
	INN         string  `json:"inn"`
	BranchID    int     `json:"branch_id"`
}

type DashboardSearchResult struct {
	CompanyID   int64  `json:"company_id"`   // ID компании (для клика по карточке)
	Number      string `json:"number"`       // Автоматический номер (№1, №2...)
	CompanyName string `json:"company_name"` // Название ООО (ҶДММ)
}
