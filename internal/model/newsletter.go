package model

import "time"

type Subscriber struct {
	ID           uint      `gorm:"primaryKey"`
	UserID       uint      `gorm:"not null"`       // Form sahibinin User ID'si
	Email        string    `gorm:"not null"`       // Abonenin e-posta adresi
	SubscribedAt time.Time `gorm:"autoCreateTime"` // Abonelik zamanı
}
