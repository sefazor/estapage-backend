package controller

import (
	"encoding/json"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stripe/stripe-go/v74"
	"github.com/stripe/stripe-go/v74/customer"
	"github.com/stripe/stripe-go/v74/subscription"
	"github.com/stripe/stripe-go/v74/webhook"

	"estepage_backend/internal/model"
	"estepage_backend/pkg/database"
	"estepage_backend/pkg/utils/jwt"
)

type SubscriptionInput struct {
	PlanID string `json:"plan_id" validate:"required"`
}

// ListPlans abonelik planlarını listeler
func ListPlans(c *fiber.Ctx) error {
	var plans []model.Subscription
	if err := database.DB.Find(&plans).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not fetch subscription plans",
		})
	}

	return c.JSON(plans)
}

// Subscribe kullanıcıyı bir plana abone yapar
func Subscribe(c *fiber.Ctx) error {
	input := new(SubscriptionInput)
	if err := c.BodyParser(input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid input",
		})
	}

	claims := c.Locals("user").(*jwt.Claims)

	// Planı kontrol et
	var plan model.Subscription
	if err := database.DB.First(&plan, "stripe_price_id = ?", input.PlanID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Subscription plan not found",
		})
	}

	// Kullanıcıyı getir
	var user model.User
	if err := database.DB.First(&user, claims.UserID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	// Stripe API key'i ayarla
	stripe.Key = os.Getenv("STRIPE_SECRET_KEY")

	// Stripe müşterisi oluştur
	customerParams := &stripe.CustomerParams{
		Email: stripe.String(user.Email),
		Name:  stripe.String(user.CompanyName),
	}

	stripeCustomer, err := customer.New(customerParams)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not create Stripe customer",
		})
	}

	// Stripe aboneliği oluştur
	subscriptionParams := &stripe.SubscriptionParams{
		Customer: stripe.String(stripeCustomer.ID),
		Items: []*stripe.SubscriptionItemsParams{
			{
				Price: stripe.String(plan.StripePriceID),
			},
		},
	}

	stripeSubscription, err := subscription.New(subscriptionParams)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not create subscription",
		})
	}

	// Unix timestamp'i tarih string'ine çevir
	expiresAt := time.Unix(stripeSubscription.CurrentPeriodEnd, 0).Format(time.RFC3339)

	// Veritabanına aboneliği kaydet
	userSubscription := model.UserSubscription{
		UserID:         claims.UserID,
		SubscriptionID: plan.ID,
		Status:         "active",
		StripeSubID:    stripeSubscription.ID,
		ExpiresAt:      expiresAt,
	}

	if err := database.DB.Create(&userSubscription).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not save subscription",
		})
	}

	return c.JSON(fiber.Map{
		"message":      "Subscription created successfully",
		"subscription": userSubscription,
	})
}

// CancelSubscription aboneliği iptal eder
func CancelSubscription(c *fiber.Ctx) error {
	claims := c.Locals("user").(*jwt.Claims)

	var userSub model.UserSubscription
	if err := database.DB.Where("user_id = ? AND status = ?", claims.UserID, "active").
		First(&userSub).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "No active subscription found",
		})
	}

	// Stripe'da aboneliği iptal et
	stripe.Key = os.Getenv("STRIPE_SECRET_KEY")
	_, err := subscription.Cancel(userSub.StripeSubID, nil)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not cancel Stripe subscription",
		})
	}

	// Veritabanında aboneliği güncelle
	userSub.Status = "cancelled"
	if err := database.DB.Save(&userSub).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not update subscription status",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Subscription cancelled successfully",
	})
}

// GetMySubscription kullanıcının aktif aboneliğini getirir
func GetMySubscription(c *fiber.Ctx) error {
	claims := c.Locals("user").(*jwt.Claims)

	var userSub model.UserSubscription
	if err := database.DB.Where("user_id = ? AND status = ?", claims.UserID, "active").
		Preload("Subscription").First(&userSub).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "No active subscription found",
		})
	}

	return c.JSON(userSub)
}

// HandleStripeWebhook Stripe webhook'larını işler
func HandleStripeWebhook(c *fiber.Ctx) error {
	webhookSecret := os.Getenv("STRIPE_WEBHOOK_SECRET")

	payload := c.Body()
	signatureHeader := c.Get("Stripe-Signature")

	event, err := webhook.ConstructEvent(payload, signatureHeader, webhookSecret)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid webhook signature",
		})
	}

	switch event.Type {
	case "customer.subscription.deleted":
		var subData struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(event.Data.Raw, &subData); err != nil {
			return c.Status(fiber.StatusBadRequest).Send(nil)
		}

		// Veritabanında aboneliği güncelle
		if err := database.DB.Model(&model.UserSubscription{}).
			Where("stripe_sub_id = ?", subData.ID).
			Update("status", "cancelled").Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Could not update subscription status",
			})
		}

	case "customer.subscription.updated":
		var subData struct {
			ID               string `json:"id"`
			CurrentPeriodEnd int64  `json:"current_period_end"`
		}
		if err := json.Unmarshal(event.Data.Raw, &subData); err != nil {
			return c.Status(fiber.StatusBadRequest).Send(nil)
		}

		expiresAt := time.Unix(subData.CurrentPeriodEnd, 0).Format(time.RFC3339)

		// Veritabanında aboneliği güncelle
		if err := database.DB.Model(&model.UserSubscription{}).
			Where("stripe_sub_id = ?", subData.ID).
			Update("expires_at", expiresAt).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Could not update subscription expiry",
			})
		}
	}

	return c.SendStatus(fiber.StatusOK)
}
