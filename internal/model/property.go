package model

import (
	"strings"

	"gorm.io/gorm"
)

// Property Types
type PropertyType string

const (
	PropertyTypeHouse      PropertyType = "House"
	PropertyTypeApartment  PropertyType = "Apartment"
	PropertyTypeCondo      PropertyType = "Condo"
	PropertyTypeVilla      PropertyType = "Villa"
	PropertyTypeTownhouse  PropertyType = "Townhouse"
	PropertyTypeLand       PropertyType = "Land"
	PropertyTypeCommercial PropertyType = "Commercial"
	PropertyTypeIndustrial PropertyType = "Industrial"
)

// Property Status
type PropertyStatus string

const (
	PropertyStatusForSale       PropertyStatus = "For Sale"
	PropertyStatusForRent       PropertyStatus = "For Rent"
	PropertyStatusSold          PropertyStatus = "Sold"
	PropertyStatusRented        PropertyStatus = "Rented"
	PropertyStatusUnderContract PropertyStatus = "Under Contract"
)

// Currency Types
type Currency string

const (
	CurrencyUSD Currency = "USD"
	CurrencyEUR Currency = "EUR"
	CurrencyTRY Currency = "TRY"
	CurrencyGBP Currency = "GBP"
	CurrencyJPY Currency = "JPY"
	CurrencyAUD Currency = "AUD"
	CurrencyCAD Currency = "CAD"
)

type Property struct {
	gorm.Model
	Title       string         `json:"title" gorm:"not null"`
	Slug        string         `json:"slug" gorm:"uniqueIndex:idx_user_property_slug;not null"`
	Type        PropertyType   `json:"type" gorm:"not null"`
	Status      PropertyStatus `json:"status" gorm:"not null"`
	Price       float64        `json:"price" gorm:"not null"`
	Currency    Currency       `json:"currency" gorm:"not null"`
	Description string         `json:"description" gorm:"type:text"`

	UserID uint `json:"user_id" gorm:"uniqueIndex:idx_user_property_slug"`

	// Location fields
	CountryCode string `json:"country_code" gorm:"not null"`
	CountryName string `json:"country_name" gorm:"not null"`
	StateCode   string `json:"state_code" gorm:"not null"`
	StateName   string `json:"state_name" gorm:"not null"`
	City        string `json:"city" gorm:"not null"`
	District    string `json:"district"`
	FullAddress string `json:"full_address" gorm:"type:text"`

	// Features fields

	Bedrooms        int  `json:"bedrooms"gorm:"not null"`         // Yatak odası sayısı
	Bathrooms       int  `json:"bathrooms"gorm:"not null"`        // Banyo sayısı
	GarageSpaces    int  `json:"garage_spaces"gorm:"not null"`    // Garaj alanı
	AreaSqFt        int  `json:"area_sq_ft"gorm:"not null"`       // Alan (sq ft)
	YearBuilt       int  `json:"year_built"gorm:"not null"`       // İnşa yılı
	SwimmingPool    bool `json:"swimming_pool"gorm:"not null"`    // Havuz var mı?
	Garden          bool `json:"garden"`                          // Bahçe var mı?
	AirConditioning bool `json:"air_conditioning"gorm:"not null"` // Klima var mı?
	CentralHeating  bool `json:"central_heating"gorm:"not null"`  // Merkezi ısıtma var mı?
	SecuritySystem  bool `json:"security_system"gorm:"not null"`  // Güvenlik sistemi var mı?

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
