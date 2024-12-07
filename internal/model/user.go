package model

import (
	"estepage_backend/pkg/database"
	"estepage_backend/pkg/subscription"
	"strings"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Email    string `gorm:"uniqueIndex;not null"`
	Password string `gorm:"not null"`
	Username string `gorm:"uniqueIndex;not null"`

	// Opsiyonel profil bilgileri (settings'den güncellenecek)

	// Kişisel Bilgiler
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	PhoneNumber string `json:"phone_number"`

	// Şirket ve Görünür bilgileri
	Title          string `json:"title"`
	BusinessEmail  string `json:"business_email"`
	WhatsAppNumber string `json:"whats_app_number"`
	Avatar         string `json:"avatar"`
	AboutMe        string `json:"about_me"`
	CompanyName    string `json:"company_name" gorm:"not null"`

	// Profesyonel İstatistikler
	Experience   int     `json:"experience"`
	TotalClients uint    `json:"total_clients"`
	SoldScore    int     `json:"sold_score"`
	Rating       float64 `json:"rating"`

	// Social Media
	SocialLinks datatypes.JSON `json:"social_links"`

	// Sistem bilgileri
	IsVerified     bool  `json:"is_verified" gorm:"default:false"`
	SubscriptionID *uint `json:"subscription_id"`

	// İlişkiler
	Properties   []Property    `json:"-"`
	Subscription *Subscription `json:"-" gorm:"foreignKey:SubscriptionID"`

	// Password reset için yeni alanlar
	PasswordResetToken string `gorm:"index"`
	ResetTokenExpires  time.Time
}

func (u *User) GetFullName() string {
	return strings.TrimSpace(u.FirstName + " " + u.LastName)
}

func (u *User) GetPublicProfile() map[string]interface{} {
	// Aktif subscription'ı bul
	var activeSubscription UserSubscription
	database.DB.Where("user_id = ? AND status = ?", u.ID, "active").First(&activeSubscription)

	// Plan tipini belirle
	planType := subscription.DeterminePlanType(activeSubscription.StripePlanID)

	// Kullanım istatistikleri
	var listingCount int64
	database.DB.Model(&Property{}).Where("user_id = ?", u.ID).Count(&listingCount)

	var leadCount int64
	database.DB.Model(&Lead{}).Where("property_user_id = ?", u.ID).Count(&leadCount)

	var profileViews int64
	database.DB.Model(&PropertyView{}).Where("property_user_id = ?", u.ID).Count(&profileViews)

	// Plan limitleri
	planLimits := subscription.GetPlanLimits(planType)

	return map[string]interface{}{
		// Basic Info
		"id":           u.ID,
		"email":        u.Email,
		"username":     u.Username,
		"company_name": u.CompanyName,
		"first_name":   u.FirstName,
		"last_name":    u.LastName,
		"full_name":    u.GetFullName(),
		"avatar":       u.Avatar,

		// Contact Info
		"title":            u.Title,
		"phone_number":     u.PhoneNumber,
		"business_email":   u.BusinessEmail,
		"whats_app_number": u.WhatsAppNumber,

		// Professional Info
		"about_me":      u.AboutMe,
		"experience":    u.Experience,
		"total_clients": u.TotalClients,
		"sold_score":    u.SoldScore,
		"rating":        u.Rating,
		"social_links":  u.SocialLinks,

		// Account Status
		"is_verified": u.IsVerified,
		"created_at":  u.CreatedAt,

		// Subscription Info
		"subscription": map[string]interface{}{
			"plan":       string(planType),
			"status":     activeSubscription.Status,
			"expires_at": activeSubscription.ExpiresAt,
			"is_active":  activeSubscription.Status == "active",
			"limits": map[string]interface{}{
				"max_listings":       planLimits.MaxListings,
				"remaining_listings": planLimits.MaxListings - int(listingCount),
				"max_images":         planLimits.MaxImagesPerList,
				"allowed_features":   planLimits.AllowedFeatures,
			},
		},

		// Usage Stats
		"stats": map[string]interface{}{
			"listing_count": listingCount,
			"lead_count":    leadCount,
			"profile_views": profileViews,
		},

		// Profile Completion
		"profile_completion": calculateProfileCompletion(u),
	}
}

// Profile completion helper
type ProfileCompletion struct {
	Percentage    int      `json:"percentage"`
	MissingFields []string `json:"missing_fields"`
}

func calculateProfileCompletion(u *User) ProfileCompletion {
	missingFields := []string{}
	totalFields := 0
	completedFields := 0

	// Kontrol edilecek alanlar
	fields := map[string]string{
		"avatar":         u.Avatar,
		"first_name":     u.FirstName,
		"last_name":      u.LastName,
		"phone_number":   u.PhoneNumber,
		"business_email": u.BusinessEmail,
		"about_me":       u.AboutMe,
		"title":          u.Title,
	}

	for fieldName, value := range fields {
		totalFields++
		if value != "" {
			completedFields++
		} else {
			missingFields = append(missingFields, fieldName)
		}
	}

	// Sosyal medya hesapları varsa +1
	if len(u.SocialLinks) > 0 {
		totalFields++
		completedFields++
	} else {
		missingFields = append(missingFields, "social_links")
	}

	percentage := (completedFields * 100) / totalFields

	return ProfileCompletion{
		Percentage:    percentage,
		MissingFields: missingFields,
	}
}
