package model

import (
	"time"

	"gorm.io/gorm"
)

// PropertyView tekil görüntülenme kaydı
type PropertyView struct {
	gorm.Model
	PropertyID uint      `json:"property_id" gorm:"index"`      // Property ilişkisi için
	UserID     *uint     `json:"user_id" gorm:"index"`          // Giriş yapmış kullanıcı için (opsiyonel)
	IP         string    `json:"ip" gorm:"index"`               // IP bazlı kontrol için
	SessionID  string    `json:"session_id" gorm:"index"`       // Browser session'ı için
	UserAgent  string    `json:"user_agent"`                    // Browser/Device bilgisi
	ViewedAt   time.Time `json:"viewed_at" gorm:"index"`        // Görüntülenme zamanı
	IsUnique   bool      `json:"is_unique" gorm:"default:true"` // Tekil görüntülenme mi?

	// İlişkiler
	Property Property `json:"-" gorm:"foreignKey:PropertyID"`
	User     *User    `json:"-" gorm:"foreignKey:UserID"`
}

// PropertyStats genel istatistikler
type PropertyStats struct {
	gorm.Model
	PropertyID       uint      `json:"property_id" gorm:"uniqueIndex"`
	TotalViews       int64     `json:"total_views"`        // Toplam görüntülenme
	UniqueViews      int64     `json:"unique_views"`       // Tekil görüntülenme
	DailyViews       int64     `json:"daily_views"`        // Günlük görüntülenme
	WeeklyViews      int64     `json:"weekly_views"`       // Haftalık görüntülenme
	MonthlyViews     int64     `json:"monthly_views"`      // Aylık görüntülenme
	LastUpdated      time.Time `json:"last_updated"`       // Son güncelleme zamanı
	LastDailyReset   time.Time `json:"last_daily_reset"`   // Son günlük sıfırlama
	LastWeeklyReset  time.Time `json:"last_weekly_reset"`  // Son haftalık sıfırlama
	LastMonthlyReset time.Time `json:"last_monthly_reset"` // Son aylık sıfırlama

	// İlişkiler
	Property Property `json:"-" gorm:"foreignKey:PropertyID"`
}

// BeforeCreate yeni görüntülenme kaydı oluşturulmadan önce çalışır
func (pv *PropertyView) BeforeCreate(tx *gorm.DB) error {
	// Son 24 saat içinde aynı IP'den görüntüleme var mı kontrol et
	var count int64
	tx.Model(&PropertyView{}).
		Where("property_id = ? AND ip = ? AND viewed_at > ?",
			pv.PropertyID,
			pv.IP,
			time.Now().Add(-24*time.Hour)).
		Count(&count)

	// Eğer son 24 saat içinde görüntüleme varsa, tekil değil
	if count > 0 {
		pv.IsUnique = false
	}

	return nil
}

// AfterCreate yeni görüntülenme kaydı oluşturulduktan sonra çalışır
func (pv *PropertyView) AfterCreate(tx *gorm.DB) error {
	var stats PropertyStats
	tx.FirstOrCreate(&stats, PropertyStats{PropertyID: pv.PropertyID})

	// İstatistikleri güncelle
	updates := map[string]interface{}{
		"total_views":   gorm.Expr("total_views + ?", 1),
		"daily_views":   gorm.Expr("daily_views + ?", 1),
		"weekly_views":  gorm.Expr("weekly_views + ?", 1),
		"monthly_views": gorm.Expr("monthly_views + ?", 1),
		"last_updated":  time.Now(),
	}

	// Eğer tekil görüntülenmeyse unique_views'i artır
	if pv.IsUnique {
		updates["unique_views"] = gorm.Expr("unique_views + ?", 1)
	}

	return tx.Model(&stats).Updates(updates).Error
}
