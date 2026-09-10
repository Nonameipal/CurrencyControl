package dto

import "time"

type CreateBranchRequest struct {
	ID   int    `json:"id" example:"5900"`
	Name string `json:"name" example:"Филиал Исмоили Сомони"`
}

type UpdateBranchRequest struct {
	Name string `json:"name" example:"Филиал Исмоили Сомони"`
}

type BranchResponse struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	CreatedBy string    `json:"created_by,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
