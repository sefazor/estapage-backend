package controller

import (
	"estepage_backend/internal/model"
	"estepage_backend/pkg/database"
	"estepage_backend/pkg/utils/jwt"
	"fmt"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// DashboardStats genel dashboard istatistikleri
type DashboardStats struct {
	TotalListings     int64              `json:"total_listings"`
	ActiveListings    int64              `json:"active_listings"`
	TotalViews        int64              `json:"total_views"`
	TopProperties     []TopProperty      `json:"top_properties"`
	DailyStats        []DailyStat        `json:"daily_stats"`
	PropertyTypeStats []PropertyTypeStat `json:"property_type_stats"`
}

type TopProperty struct {
	ID         uint    `json:"id"`
	Title      string  `json:"title"`
	Views      int64   `json:"views"`
	Price      float64 `json:"price"`
	Location   string  `json:"location"`
	Type       string  `json:"type"`
	CoverImage string  `json:"cover_image"`
}

type DailyStat struct {
	Date        string `json:"date"`
	Views       int64  `json:"views"`
	NewListings int64  `json:"new_listings"`
}

type PropertyTypeStat struct {
	Type  string `json:"type"`
	Count int64  `json:"count"`
	Views int64  `json:"views"`
}

const (
	ViewCooldown = 24 * time.Hour // Aynı IP için bekleme süresi
)

// GetDashboardStats dashboard istatistiklerini getirir
func GetDashboardStats(c *fiber.Ctx) error {
	claims := c.Locals("user").(*jwt.Claims)
	db := database.GetDB()

	var stats DashboardStats

	// Toplam ve aktif ilan sayısı
	db.Model(&model.Property{}).
		Where("user_id = ?", claims.UserID).
		Count(&stats.TotalListings)

	db.Model(&model.Property{}).
		Where("user_id = ? AND status = ?", claims.UserID, "active").
		Count(&stats.ActiveListings)

	// Toplam görüntülenme
	db.Model(&model.PropertyView{}).
		Joins("JOIN properties ON property_views.property_id = properties.id").
		Where("properties.user_id = ?", claims.UserID).
		Count(&stats.TotalViews)

	// En çok görüntülenen 5 ilan
	var topProps []TopProperty
	db.Table("properties").
		Select("properties.id, properties.title, properties.price, properties.location, properties.type, COUNT(property_views.id) as views").
		Joins("LEFT JOIN property_views ON properties.id = property_views.property_id").
		Where("properties.user_id = ? AND properties.status = ?", claims.UserID, "active").
		Group("properties.id").
		Order("views DESC").
		Limit(5).
		Scan(&topProps)

	// Her bir ilan için kapak fotoğrafını ekle
	for i := range topProps {
		var coverImage model.PropertyImage
		db.Where("property_id = ? AND is_cover = ?", topProps[i].ID, true).
			First(&coverImage)
		topProps[i].CoverImage = coverImage.URL
	}
	stats.TopProperties = topProps

	// Son 7 günün istatistikleri
	var dailyStats []DailyStat
	for i := 6; i >= 0; i-- {
		date := time.Now().AddDate(0, 0, -i)
		var stat DailyStat
		stat.Date = date.Format("2006-01-02")

		// Günlük görüntülenme
		db.Model(&model.PropertyView{}).
			Joins("JOIN properties ON property_views.property_id = properties.id").
			Where("properties.user_id = ? AND DATE(property_views.created_at) = ?",
				claims.UserID, date.Format("2006-01-02")).
			Count(&stat.Views)

		// Günlük yeni ilan
		db.Model(&model.Property{}).
			Where("user_id = ? AND DATE(created_at) = ?",
				claims.UserID, date.Format("2006-01-02")).
			Count(&stat.NewListings)

		dailyStats = append(dailyStats, stat)
	}
	stats.DailyStats = dailyStats

	return c.JSON(stats)
}

// RecordPropertyView ilan görüntülenmesini kaydeder
func RecordPropertyView(c *fiber.Ctx) error {
	propertyIDStr := c.Params("id")
	propertyID, err := strconv.ParseUint(propertyIDStr, 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid property ID",
		})
	}

	// İlanın varlığını kontrol et
	var property model.Property
	if err := database.GetDB().First(&property, propertyID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Property not found",
		})
	}

	ip := c.IP()
	userAgent := c.Get("User-Agent")

	// Session ID'yi header'dan al veya oluştur
	sessionID := c.Get("X-Session-ID")
	if sessionID == "" {
		sessionID = fmt.Sprintf("%s_%s", ip, time.Now().Format("20060102"))
	}

	// Kullanıcı girişi varsa ID'sini al
	var userID *uint
	if claims, ok := c.Locals("user").(*jwt.Claims); ok {
		userID = &claims.UserID
	}

	// Son 24 saat içinde aynı IP'den görüntüleme var mı kontrol et
	var lastView model.PropertyView
	result := database.GetDB().Where(
		"property_id = ? AND ip = ? AND created_at > ?",
		propertyID,
		ip,
		time.Now().Add(-ViewCooldown),
	).First(&lastView)

	// Eğer son 24 saat içinde görüntüleme yoksa kaydet
	if result.RowsAffected == 0 {
		view := model.PropertyView{
			PropertyID: uint(propertyID),
			UserID:     userID,
			IP:         ip,
			SessionID:  sessionID,
			UserAgent:  userAgent,
			ViewedAt:   time.Now(),
		}

		if err := database.GetDB().Create(&view).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Could not record view",
			})
		}

		// İstatistikleri güncelle
		stats := model.PropertyStats{}
		database.GetDB().FirstOrCreate(&stats, model.PropertyStats{PropertyID: uint(propertyID)})

		database.GetDB().Model(&stats).Updates(map[string]interface{}{
			"total_views":   gorm.Expr("total_views + ?", 1),
			"unique_views":  gorm.Expr("unique_views + ?", 1),
			"daily_views":   gorm.Expr("daily_views + ?", 1),
			"weekly_views":  gorm.Expr("weekly_views + ?", 1),
			"monthly_views": gorm.Expr("monthly_views + ?", 1),
		})
	}

	return c.SendStatus(fiber.StatusOK)
}
