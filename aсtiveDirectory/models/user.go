package models

type RequestUser struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

// UserInfo содержит ФИО пользователя, полученные из Active Directory
type UserInfo struct {
	LastName   string `json:"last_name"`
	FirstName  string `json:"first_name"`
	MiddleName string `json:"middle_name"`
	Email      string `json:"email"`
	Department string `json:"department"`
	Position   string `json:"position"`
	Phone      string `json:"phone"`
}
