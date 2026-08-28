package dto

type CreateCompanyRequest struct {
	LLC      string `json:"llc"`
	INN      string `json:"inn"`
}

type CompanyResponse struct {
	ID       int64  `json:"id"`
	LLC      string `json:"llc"`
	INN      string `json:"inn"`
	BranchID int    `json:"branch_id"`
}
