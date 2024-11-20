package model

import (
	"estepage_backend/pkg/database"
	"time"

	"gorm.io/gorm"
)

type Lead struct {
	gorm.Model
	PropertyID  uint       `json:"property_id" gorm:"index"`
	Name        string     `json:"name"`
	Email       string     `json:"email"`
	Phone       string     `json:"phone"`
	Message     string     `json:"message" gorm:"type:text"`
	Status      string     `json:"status" gorm:"default:'new'"` // new, contacted, qualified, converted, closed
	ReadStatus  bool       `json:"read_status" gorm:"default:false"`
	ContactedAt *time.Time `json:"contacted_at"`

	// İlişkiler
	Property Property `json:"property" gorm:"foreignKey:PropertyID"`
}

// Property modelini de ayrıca güncelleyelim
func (p *Property) GetLeadsCount() (int64, error) {
	var count int64
	db := database.GetDB()
	err := db.Model(&Lead{}).Where("property_id = ?", p.ID).Count(&count).Error
	return count, err
}
