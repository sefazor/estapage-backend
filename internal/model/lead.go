package model

import (
	"estepage_backend/pkg/database"
	"gorm.io/gorm"
)


type LeadStatus string
const (
    LeadStatusNew        LeadStatus = "new"
    LeadStatusRead       LeadStatus = "read"
    LeadStatusContacted  LeadStatus = "contacted"
    LeadStatusNoResponse LeadStatus = "no_response"
    LeadStatusCompleted  LeadStatus = "completed"
)


type Lead struct {
	gorm.Model
	PropertyID  uint       `json:"property_id" gorm:"index"`
	Name        string     `json:"name"`
	Email       string     `json:"email"`
	Phone       string     `json:"phone"`
	Message     string     `json:"message" gorm:"type:text"`
	ReadStatus  bool       `json:"read_status" gorm:"default:false"`
	Status      LeadStatus `json:"status" gorm:"type:string;default:'new'"`
	Property Property `json:"property"`
}

// Property modelini de ayrıca güncelleyelim
func (p *Property) GetLeadsCount() (int64, error) {
	var count int64
	db := database.GetDB()
	err := db.Model(&Lead{}).Where("property_id = ?", p.ID).Count(&count).Error
	return count, err
}
