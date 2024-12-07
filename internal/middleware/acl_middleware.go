// acl_middleware.go
package middleware

import (
	"estepage_backend/internal/model"
	"estepage_backend/pkg/database"
	"estepage_backend/pkg/subscription"
	"estepage_backend/pkg/utils/jwt"

	"github.com/gofiber/fiber/v2"
)

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

func CheckSubscriptionLimit() fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims := c.Locals("user").(*jwt.Claims)

		// Aktif abonelik kontrolü
		var activeSubscription model.UserSubscription
		planType := subscription.FreePlan // Default olarak Free plan

		if err := database.DB.Where("user_id = ? AND status = ?", claims.UserID, "active").
			First(&activeSubscription).Error; err == nil {
			// Aktif abonelik varsa plan tipini belirle
			planType = subscription.DeterminePlanType(activeSubscription.StripePlanID)
		}

		// İlan sayısı kontrolü
		var propertyCount int64
		if err := database.DB.Model(&model.Property{}).Where("user_id = ?", claims.UserID).Count(&propertyCount).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Could not check property count",
			})
		}

		// Plan limitlerini kontrol et
		planLimits := subscription.GetPlanLimits(planType)
		if int(propertyCount) >= planLimits.MaxListings {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error":         "You have reached your property listing limit",
				"current_count": propertyCount,
				"max_limit":     planLimits.MaxListings,
				"plan":          planType,
			})
		}

		return c.Next()
	}
}

// subscription.go
func CheckFeatureAccess(feature subscription.Feature) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims := c.Locals("user").(*jwt.Claims)

		// Aktif abonelik kontrolü
		var activeSubscription model.UserSubscription
		planType := subscription.FreePlan

		if err := database.DB.Where("user_id = ? AND status = ?", claims.UserID, "active").
			First(&activeSubscription).Error; err == nil {
			planType = subscription.DeterminePlanType(activeSubscription.StripePlanID)
		}

		// Feature kullanım kontrolü
		if !subscription.CanUseFeature(planType, feature) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error":        "This feature requires a higher subscription plan",
				"current_plan": planType,
				"feature":      feature,
			})
		}

		return c.Next()
	}
}

func CheckImageLimit() fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims := c.Locals("user").(*jwt.Claims)
		propertyID := c.Params("property_id")

		// Aktif abonelik kontrolü
		var activeSubscription model.UserSubscription
		planType := subscription.FreePlan

		if err := database.DB.Where("user_id = ? AND status = ?", claims.UserID, "active").
			First(&activeSubscription).Error; err == nil {
			planType = subscription.DeterminePlanType(activeSubscription.StripePlanID)
		}

		// Mevcut resim sayısını kontrol et
		var imageCount int64
		if err := database.DB.Model(&model.PropertyImage{}).
			Where("property_id = ?", propertyID).Count(&imageCount).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Could not check image count",
			})
		}

		// Plan limitlerini kontrol et
		planLimits := subscription.GetPlanLimits(planType)
		if int(imageCount) >= planLimits.MaxImagesPerList {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error":         "You have reached the maximum image limit for this listing",
				"current_count": imageCount,
				"max_limit":     planLimits.MaxImagesPerList,
				"plan":          planType,
			})
		}

		return c.Next()
	}
}
