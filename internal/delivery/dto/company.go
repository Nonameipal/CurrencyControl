package dto

type CreateCompanyRequest struct {
	Name     string `json:"name"`
	INN      string `json:"inn"`
	BranchID int    `json:"branch_id"`
}

type CompanyResponse struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	INN      string `json:"inn"`
	BranchID int    `json:"branch_id"`
}
