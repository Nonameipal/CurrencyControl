package dto

type DashboardSearchRequest struct {
	CompanyName string  `json:"company_name"`
	Amount      float64 `json:"amount"`
	INN         string  `json:"inn"`
	BranchID    int     `json:"branch_id"`
}

type DashboardSearchResult struct {
	CompanyID   int64  `json:"company_id"`   
	Number      string `json:"number"`       
	CompanyName string `json:"company_name"` 
	INN         string `json:"inn"`
}
