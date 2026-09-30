package domain
import (
	"time"
)
type CurrencyControlPermission struct {
	Login     string     `gorm:"primaryKey" db:"login" json:"login"`
	CanCreate bool       `gorm:"not null;default:false" db:"can_create" json:"can_create"`
	CanEdit   bool       `gorm:"not null;default:true" db:"can_edit" json:"can_edit"`
	CanDelete bool       `gorm:"not null;default:true" db:"can_delete" json:"can_delete"`
	GrantedBy string     `gorm:"not null" db:"granted_by" json:"granted_by"`
	GrantedAt time.Time  `gorm:"not null;default:now()" db:"granted_at" json:"granted_at"`
	User      *UserBrief `gorm:"-" json:"user,omitempty"`
	Granter   *UserBrief `gorm:"-" json:"granter,omitempty"`
}
