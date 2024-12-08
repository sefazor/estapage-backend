package model

import (
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type PropertyFeature struct {
	gorm.Model
	PropertyID uint           `json:"property_id" gorm:"index"`
	Title      string         `json:"title" gorm:"not null"`  // Özellik başlığı (örn: "Balkon Sayısı")
	Values     datatypes.JSON `json:"values"`                 // Esnek değer alanı (array veya string olabilir)
	Order      int            `json:"order" gorm:"default:0"` // Sıralama için

	// İlişki
	Property Property `json:"-" gorm:"foreignKey:PropertyID"`
}
