package model

import "time"

type NewsletterSubscriber struct {
	ID           uint      `gorm:"primaryKey"`
	UserID       uint      `gorm:"not null"`       // Form sahibinin User ID'si
	Name         string    `gorm:"size:255"`       // Abonenin adı soyadı
	Email        string    `gorm:"not null"`       // Abonenin e-posta adresi
	Source       string    `gorm:"size:50"`        // Kaynak (Property Page, Newsletter Form vs)
	SubscribedAt time.Time `gorm:"autoCreateTime"` // Abonelik zamanı
}

// Tablo adını özelleştir
func (NewsletterSubscriber) TableName() string {
	return "newsletter_subscribers"
}
