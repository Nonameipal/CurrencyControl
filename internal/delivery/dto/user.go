package dto

type CurrencyControlUserItem struct {
	ID         int64  `json:"id"`
	Login      string `json:"login"`
	FirstName  string `json:"first_name"`
	LastName   string `json:"last_name"`
	Email      string `json:"email,omitempty"`
	Role       string `json:"role"`
	RoleName   string `json:"role_name"`
	BranchID   int64  `json:"branch_id"`
	BranchName string `json:"branch_name,omitempty"`
}
