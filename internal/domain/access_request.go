package domain
import (
	"time"
)
type AccessRequest struct {
	ID           int64      `gorm:"primaryKey"                   json:"id"`
	Login        string     `gorm:"not null;index"               json:"login"`
	LastName     string     `gorm:"type:varchar(255);default:''" json:"last_name,omitempty"`
	FirstName    string     `gorm:"type:varchar(255);default:''" json:"first_name,omitempty"`
	Email        string     `gorm:"type:varchar(255);default:''" json:"email,omitempty"`
	BranchID     int64      `gorm:"not null"                     json:"branch_id"`
	BranchName   string     `gorm:"-"                            json:"branch_name,omitempty"`
	Role         string     `gorm:"not null;default:'operator'"  json:"role"`
	Status       string     `gorm:"not null;default:'pending'"   json:"status"`
	SessionToken *string    `gorm:"uniqueIndex"                  json:"session_token,omitempty"`
	CreatedAt    time.Time  `gorm:"not null;default:now()"       json:"created_at"`
	ReviewedAt   *time.Time `json:"reviewed_at,omitempty"`
	ReviewedBy   string     `gorm:"type:varchar(255);default:''" json:"reviewed_by,omitempty"`
	Applicant    *UserBrief `gorm:"-"                            json:"applicant,omitempty"`
	Reviewer     *UserBrief `gorm:"-"                            json:"reviewer,omitempty"`
}

