package middleware

import (
	"estepage_backend/internal/model"
	"estepage_backend/pkg/database"
	"estepage_backend/pkg/utils/jwt"

	"github.com/gofiber/fiber/v2"
)

// CheckPropertyOwnership emlak ilanının sahibi olup olmadığını kontrol eder
func CheckPropertyOwnership() fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims := c.Locals("user").(*jwt.Claims)
		propertyID := c.Params("id")

		var property model.Property
		if err := database.DB.First(&property, propertyID).Error; err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Property not found",
			})
		}

		if property.UserID != claims.UserID {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "You don't have permission to access this property",
			})
		}

		return c.Next()
	}
}

// CheckSubscriptionLimit kullanıcının abonelik limitini kontrol eder
func CheckSubscriptionLimit() fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims := c.Locals("user").(*jwt.Claims)

		var user model.User
		if err := database.DB.Preload("UserSubscriptions").First(&user, claims.UserID).Error; err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "User not found",
			})
		}

		// Aktif abonelik kontrolü
		var activeSubscription model.UserSubscription
		if err := database.DB.Where("user_id = ? AND status = ?", claims.UserID, "active").
			First(&activeSubscription).Error; err != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "No active subscription found",
			})
		}

		// İlan sayısı kontrolü
		var propertyCount int64
		database.DB.Model(&model.Property{}).Where("user_id = ?", claims.UserID).Count(&propertyCount)

		var subscription model.Subscription
		if err := database.DB.First(&subscription, activeSubscription.SubscriptionID).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Could not fetch subscription details",
			})
		}

		if int(propertyCount) >= subscription.MaxListings {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error":         "You have reached your property listing limit",
				"current_count": propertyCount,
				"max_limit":     subscription.MaxListings,
			})
		}

		return c.Next()
	}
}
