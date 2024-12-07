package middleware

import (
	"estepage_backend/internal/model"
	"estepage_backend/pkg/database"
	"estepage_backend/pkg/subscription"
	"estepage_backend/pkg/utils/jwt"

	"github.com/gofiber/fiber/v2"
)

func CheckSubscriptionFeature(feature subscription.Feature) fiber.Handler {
	return func(c *fiber.Ctx) error {
		claims := c.Locals("user").(*jwt.Claims)

		// Kullanıcının aktif subscription'ını kontrol et
		var userSub model.UserSubscription
		var planType subscription.PlanType = subscription.FreePlan // Default olarak Free

		if err := database.DB.Where("user_id = ? AND status = ?", claims.UserID, "active").
			First(&userSub).Error; err == nil {
			// Stripe plan ID'sine göre plan tipini belirle
			planType = determinePlanType(userSub.StripePlanID)
		}

		if !subscription.CanUseFeature(planType, feature) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "This feature requires a higher subscription plan",
			})
		}

		return c.Next()
	}
}

func CheckListingLimit(c *fiber.Ctx) error {
	claims := c.Locals("user").(*jwt.Claims)

	var userSub model.UserSubscription
	planType := subscription.FreePlan

	if err := database.DB.Where("user_id = ? AND status = ?", claims.UserID, "active").
		First(&userSub).Error; err == nil {
		planType = determinePlanType(userSub.StripePlanID)
	}

	limits := subscription.GetPlanLimits(planType)

	var currentListings int64
	database.DB.Model(&model.Property{}).Where("user_id = ?", claims.UserID).Count(&currentListings)

	if int(currentListings) >= limits.MaxListings {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "You have reached your listing limit. Please upgrade your plan.",
		})
	}

	return c.Next()
}

// Stripe plan ID'sinden plan tipini belirle
func determinePlanType(stripePlanID string) subscription.PlanType {
	switch stripePlanID {
	case "price_1QT3IEJuNU9LluRUWytR6JS5":
		return subscription.ProPlan
	case "price_1QT3IaJuNU9LluRUg21Cv7QU":
		return subscription.ElitePlan
	default:
		return subscription.FreePlan
	}
}
