package model

import (
	"strings"

	"gorm.io/gorm"
)

type Property struct {
	gorm.Model
	Title       string  `json:"title" gorm:"not null"`
	Slug        string  `json:"slug" gorm:"uniqueIndex:idx_user_property_slug;not null"`
	Description string  `json:"description" gorm:"type:text"`
	Price       float64 `json:"price"`
	Location    string  `json:"location"`
	Type        string  `json:"type"`
	Size        float64 `json:"size"`
	Rooms       int     `json:"rooms"`
	Bathrooms   int     `json:"bathrooms"`
	Features    string  `json:"features" gorm:"type:text"`
	Status      string  `json:"status" gorm:"default:'active'"`
	UserID      uint    `json:"user_id" gorm:"uniqueIndex:idx_user_property_slug"`
	VideoURL    *string `json:"video_url" gorm:"type:text"` // Tek video URL'i

	// İlişkiler
	User   User            `json:"-" gorm:"foreignKey:UserID"`
	Images []PropertyImage `json:"images" gorm:"foreignKey:PropertyID;constraint:OnDelete:CASCADE"`
}

type PropertyImage struct {
	gorm.Model
	PropertyID uint   `json:"property_id"`
	URL        string `json:"url" gorm:"not null"`
	IsCover    bool   `json:"is_cover" gorm:"default:false"`
	Order      int    `json:"order" gorm:"default:0"`

	Property Property `json:"-" gorm:"foreignKey:PropertyID"`
}

// BeforeCreate property oluşturulurken slug'ı otomatik oluşturur
func (p *Property) BeforeCreate(tx *gorm.DB) error {
	if p.Slug == "" {
		slug := strings.ToLower(strings.ReplaceAll(p.Title, " ", "-"))
		// Özel karakterleri temizle
		slug = strings.Map(func(r rune) rune {
			if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
				return r
			}
			return -1
		}, slug)

		// Slug'ın benzersiz olduğundan emin ol
		var count int64
		tx.Model(&Property{}).Where("user_id = ? AND slug = ?", p.UserID, slug).Count(&count)
		if count > 0 {
			// Eğer aynı slug varsa sonuna timestamp ekle
			slug = slug + "-" + p.CreatedAt.Format("20060102")
		}

		p.Slug = slug
	}
	return nil
}
