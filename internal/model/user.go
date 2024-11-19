package model

import (
	"strings"

	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Email       string `gorm:"uniqueIndex;not null"`
	Password    string `gorm:"not null"`
	CompanyName string `gorm:"not null"`
	Username    string `gorm:"uniqueIndex;not null"` // Yeni alan
	Properties  []Property
}

// BeforeCreate kullanıcı oluşturulurken username'i otomatik oluşturur
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.Username == "" {
		// CompanyName'den username oluştur
		username := strings.ToLower(strings.ReplaceAll(u.CompanyName, " ", "-"))
		// Özel karakterleri temizle
		username = strings.Map(func(r rune) rune {
			if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
				return r
			}
			return -1
		}, username)

		// Username'in benzersiz olduğundan emin ol
		var count int64
		tx.Model(&User{}).Where("username = ?", username).Count(&count)
		if count > 0 {
			username = username + "-" + strings.Split(u.Email, "@")[0]
		}

		u.Username = username
	}
	return nil
}
