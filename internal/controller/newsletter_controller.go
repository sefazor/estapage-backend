package controller

import (
	"estepage_backend/internal/model"
	"estepage_backend/pkg/database"
	"estepage_backend/pkg/utils/jwt"
	"net/mail"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

type NewsletterSubscriptionInput struct {
	Name  string `json:"name"`
	Email string `json:"email" validate:"required,email"`
}

const (
	SourcePropertyPage   = "Property Page"
	SourceProfilePage    = "Profile Page"
	SourceNewsletterForm = "Newsletter Form"
)

func determineSubscriptionSource(c *fiber.Ctx) string {
	referer := c.Get("Referer")

	// URL path'lerini parse et
	pathParts := strings.Split(referer, "/")

	if len(pathParts) >= 2 {
		// Property sayfası kontrolü
		// Örnek: /p/username veya /p/username/property-slug
		if pathParts[1] == "p" {
			return SourcePropertyPage
		}

		// Profile sayfası kontrolü
		// Örnek: /username (root path'deki kullanıcı profili)
		if len(pathParts) == 2 && pathParts[1] != "" {
			return SourceProfilePage
		}
	}

	// Eğer başka bir endpoint'den geliyorsa (örn. form)
	if strings.Contains(c.Path(), "/newsletter/") {
		return SourceNewsletterForm
	}

	// Varsayılan olarak Property Page döndür
	return SourcePropertyPage
}

func PublicSubscribe(c *fiber.Ctx) error {
	userIDStr := c.Params("user_id")
	if userIDStr == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Agent ID is required",
		})
	}

	userID, err := strconv.ParseUint(userIDStr, 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid agent ID format",
		})
	}

	var input NewsletterSubscriptionInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid input format",
		})
	}

	if _, err := mail.ParseAddress(input.Email); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid email format",
		})
	}

	var existingSubscriber model.NewsletterSubscriber
	if err := database.GetDB().Where("user_id = ? AND email = ?", userID, input.Email).
		First(&existingSubscriber).Error; err == nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "Already subscribed to this agent's newsletter",
		})
	}

	source := determineSubscriptionSource(c)

	subscriber := model.NewsletterSubscriber{
		UserID: uint(userID),
		Name:   input.Name,
		Email:  input.Email,
		Source: source,
	}

	if err := database.GetDB().Create(&subscriber).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not complete subscription",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Successfully subscribed to newsletter",
		"source":  source,
	})
}

func GetMySubscribers(c *fiber.Ctx) error {
	claims := c.Locals("user").(*jwt.Claims)

	type SubscriberResponse struct {
		ID           uint      `json:"id"`
		Name         string    `json:"name"`
		Email        string    `json:"email"`
		Source       string    `json:"source"`
		SubscribedAt time.Time `json:"join_date"`
	}

	var subscribers []SubscriberResponse

	if err := database.GetDB().Model(&model.NewsletterSubscriber{}).
		Select("id, name, email, source, subscribed_at").
		Where("user_id = ?", claims.UserID).
		Order("subscribed_at DESC").
		Find(&subscribers).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not fetch subscribers",
		})
	}

	return c.JSON(fiber.Map{
		"subscribers": subscribers,
		"total":       len(subscribers),
	})
}

// Stats
func GetNewsletterStats(c *fiber.Ctx) error {
	claims := c.Locals("user").(*jwt.Claims)

	var dailyStats []struct {
		Date  string `json:"date"`
		Count int64  `json:"count"`
	}

	err := database.GetDB().Raw(`
        SELECT 
            DATE(subscribed_at) as date,
            COUNT(*) as count
        FROM newsletter_subscribers
        WHERE user_id = ?
        AND subscribed_at >= CURRENT_DATE - INTERVAL '7 days'
        GROUP BY DATE(subscribed_at)
        ORDER BY date DESC
    `, claims.UserID).Scan(&dailyStats).Error

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not fetch newsletter statistics",
		})
	}

	// Toplam abone sayısı
	var totalSubscribers int64
	database.GetDB().Model(&model.NewsletterSubscriber{}).
		Where("user_id = ?", claims.UserID).
		Count(&totalSubscribers)

	// Son ay içindeki abone sayısı
	var monthlySubscribers int64
	database.GetDB().Model(&model.NewsletterSubscriber{}).
		Where("user_id = ? AND subscribed_at >= CURRENT_DATE - INTERVAL '30 days'", claims.UserID).
		Count(&monthlySubscribers)

	return c.JSON(fiber.Map{
		"daily_stats":         dailyStats,
		"total_subscribers":   totalSubscribers,
		"monthly_subscribers": monthlySubscribers,
	})
}
