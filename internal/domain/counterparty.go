package domain

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

const (
	ClientTypeLegalEntity = "legal_entity" // Юридическое лицо
	ClientTypeIndividual  = "individual"   // Физическое лицо
)

type Counterparty struct {
	ID         int64          `gorm:"primaryKey" db:"id" json:"id"`
	BranchID   int            `gorm:"index" db:"branch_id" json:"branch_id"`
	Name       string         `gorm:"not null" db:"name" json:"name"`
	INN        *string        `gorm:"type:varchar(22);index" db:"inn" json:"inn"`
	ClientType string         `gorm:"type:varchar(50);default:'legal_entity'" db:"client_type" json:"client_type"`
	Phones     string         `gorm:"type:text;default:'[]'" db:"phones" json:"-"`
	Accounts   string         `gorm:"type:text;default:'[]'" db:"accounts" json:"-"`
	Operator   string         `gorm:"type:varchar(255);default:''" db:"operator" json:"operator"`
	Email      string         `db:"email" json:"email"`
	CreatedBy  string         `gorm:"type:varchar(255);default:''" db:"created_by" json:"created_by"`
	CreatedAt  time.Time      `gorm:"not null;default:now()" db:"created_at" json:"created_at"`
	UpdatedAt  time.Time      `gorm:"not null;default:now()" db:"updated_at" json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	PhonesList   []string `gorm:"-" json:"phones,omitempty"`
	AccountsList []string `gorm:"-" json:"accounts,omitempty"`
}

func (c *Counterparty) GetPhones() []string {
	if len(c.PhonesList) > 0 {
		return c.PhonesList
	}
	var res []string
	if c.Phones != "" {
		_ = json.Unmarshal([]byte(c.Phones), &res)
	}
	if res == nil {
		res = []string{}
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

func (c *Counterparty) GetAccounts() []string {
	if len(c.AccountsList) > 0 {
		return c.AccountsList
	}
	var res []string
	if c.Accounts != "" {
		_ = json.Unmarshal([]byte(c.Accounts), &res)
	}
	if res == nil {
		res = []string{}
	}
	return res
}

func (c *Counterparty) SetAccounts(accounts []string) {
	if accounts == nil {
		accounts = []string{}
	}
	c.AccountsList = accounts
	b, _ := json.Marshal(accounts)
	c.Accounts = string(b)
}
