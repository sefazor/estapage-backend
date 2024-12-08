// internal/model/login_history.go
package model

import "time"

type LoginHistory struct {
	ID        uint      `gorm:"primaryKey"`
	UserID    uint      `gorm:"not null"`
	Device    string    `gorm:"size:100"` // Chrome on Windows, Safari on iPhone
	Location  string    `gorm:"size:100"` // New York, USA
	IP        string    `gorm:"size:50"`  // IP adresi
	CreatedAt time.Time `gorm:"autoCreateTime"`
}
