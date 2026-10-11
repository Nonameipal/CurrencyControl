package dto

import "time"

type AccessRequestNotificationItem struct {
	ID         int64     `json:"id"`
	Type       string    `json:"type"`
	Title      string    `json:"title"`
	Message    string    `json:"message"`
	RequestID  int64     `json:"request_id"`
	Login      string    `json:"login"`
	FullName   string    `json:"full_name"`
	Email      string    `json:"email,omitempty"`
	Role       string    `json:"role"`
	RoleName   string    `json:"role_name"`
	BranchID   int64     `json:"branch_id" `
	BranchName string    `json:"branch_name,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	Status     string    `json:"status" `
	Actions    []string  `json:"actions"`
}

type AccessRequestNotificationsResponse struct {
	Total int                             `json:"total"`
	Items []AccessRequestNotificationItem `json:"items"`
}
