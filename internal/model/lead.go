// internal/model/lead.go

package model

import (
	"estepage_backend/pkg/database"

	"gorm.io/gorm"
)

type LeadSource string
type LeadStatus string

const (
	LeadSourceProfile  LeadSource = "profile_page"
	LeadSourceProperty LeadSource = "property_page"

	LeadStatusNew        LeadStatus = "new"
	LeadStatusRead       LeadStatus = "read"
	LeadStatusContacted  LeadStatus = "contacted"
	LeadStatusNoResponse LeadStatus = "no_response"
	LeadStatusCompleted  LeadStatus = "completed"
)

type Lead struct {
	gorm.Model
	UserID     uint       `json:"user_id" gorm:"index"`
	PropertyID uint       `json:"property_id" gorm:"index;default:null"` // default null
	Source     LeadSource `json:"source"`
	Name       string     `json:"name"`
	Email      string     `json:"email"`
	Phone      string     `json:"phone"`
	Message    string     `json:"message" gorm:"type:text"`
	Status     LeadStatus `json:"status" gorm:"type:string;default:'new'"`
	ReadStatus bool       `json:"read_status" gorm:"default:false"`

	// Property detayları - hepsi nullable
	PropertyTitle    *string  `json:"property_title,omitempty"`
	PropertyPrice    *float64 `json:"property_price,omitempty"`
	PropertyImage    *string  `json:"property_image,omitempty"`
	PropertyCurrency *string  `json:"property_currency,omitempty"`

	// İlişkiler
	Property *Property `json:"property,omitempty" gorm:"foreignKey:PropertyID"`
}

// Property modelini de ayrıca güncelleyelim
func (p *Property) GetLeadsCount() (int64, error) {
	var count int64
	db := database.GetDB()
	err := db.Model(&Lead{}).Where("property_id = ?", p.ID).Count(&count).Error
	return count, err
}
