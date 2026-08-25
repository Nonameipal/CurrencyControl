package dto

type DashboardSearchRequest struct {
	CompanyName string  `json:"company_name"`
	Amount      float64 `json:"amount"`
	INN         string  `json:"inn"`
	BranchID    int     `json:"branch_id"`
}

type DashboardSearchResult struct {
	ContractID  int64  `json:"contract_id"` 
	Number      string `json:"number"`       
	CompanyName string `json:"company_name"` 
}
