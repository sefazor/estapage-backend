package model

import (
	"strings"

	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Email       string `gorm:"uniqueIndex;not null"`
	Password    string `gorm:"not null"`
	Username    string `gorm:"uniqueIndex;not null"`
	CompanyName string `json:"company_name" gorm:"not null"`

	// Opsiyonel profil bilgileri (settings'den güncellenecek)
	FirstName      string `json:"first_name"`
	LastName       string `json:"last_name"`
	Title          string `json:"title"`
	PhoneNumber    string `json:"phone_number"`
	BusinessEmail  string `json:"business_email"`
	WhatsAppNumber string `json:"whats_app_number"`
	Avatar         string `json:"avatar"`

	// Sistem bilgileri
	IsVerified     bool  `json:"is_verified" gorm:"default:false"`
	SubscriptionID *uint `json:"subscription_id"`

	// İlişkiler
	Properties   []Property    `json:"-"`
	Subscription *Subscription `json:"-" gorm:"foreignKey:SubscriptionID"`
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
		"is_verified":      u.IsVerified,
	}
}
