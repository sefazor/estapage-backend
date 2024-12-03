package model

import (
	"strings"
	"time"

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
	return map[string]interface{}{
		"id":               u.ID,
		"username":         u.Username,
		"company_name":     u.CompanyName,
		"full_name":        u.GetFullName(),
		"title":            u.Title,
		"phone_number":     u.PhoneNumber,
		"business_email":   u.BusinessEmail,
		"whats_app_number": u.WhatsAppNumber,
		"avatar":           u.Avatar,
		"about_me":         u.AboutMe,
		"experience":       u.Experience,
		"sold_score":       u.SoldScore,
		"rating":           u.Rating,
		"total_clients":    u.TotalClients,
		"is_verified":      u.IsVerified,
	}
}
