package domain

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

const (
	ClientTypeLegalEntity    = "legal_entity"
	ClientTypeIndividual     = "individual"
	ClientTypeSoleProprietor = "sole_proprietor"
)

type Counterparty struct {
	ID         int64          `gorm:"primaryKey" db:"id" json:"id"`
	BranchID   int            `gorm:"index" db:"branch_id" json:"branch_id"`
	LLC        string         `gorm:"type:varchar(500);default:''" db:"llc" json:"llc"`
	INN        *string        `gorm:"type:varchar(22);index" db:"inn" json:"inn"`
	ClientType string         `gorm:"type:varchar(50);default:'legal_entity'" db:"client_type" json:"client_type"`
	Phones     string         `gorm:"type:text;default:'[]'" db:"phones" json:"-"`
	CreatedBy  string         `gorm:"type:varchar(255);default:''" db:"created_by" json:"created_by"`
	CreatedAt  time.Time      `gorm:"not null;default:now()" db:"created_at" json:"created_at"`
	UpdatedAt  time.Time      `gorm:"not null;default:now()" db:"updated_at" json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
	UpdatedBy  string         `gorm:"type:varchar(255);default:''" db:"updated_by" json:"updated_by,omitempty"`
	DeletedBy  *string        `gorm:"type:varchar(255)" db:"deleted_by" json:"deleted_by,omitempty"`
	Creator    *UserBrief     `gorm:"-" json:"creator,omitempty"`
	Updater    *UserBrief     `gorm:"-" json:"updater,omitempty"`
	Deleter    *UserBrief     `gorm:"-" json:"deleter,omitempty"`

	PhonesList []string `gorm:"-" json:"phones,omitempty"`
}

func (c *Counterparty) GetPhones() []string {
	if len(c.PhonesList) > 0 {
		return c.PhonesList
	}
	res := []string{}
	if c.Phones != "" {
		_ = json.Unmarshal([]byte(c.Phones), &res)
	}
	return res
}

func (c *Counterparty) SetPhones(phones []string) {
	if phones == nil {
		phones = []string{}
	}
	c.PhonesList = phones
	b, _ := json.Marshal(phones)
	c.Phones = string(b)
}
