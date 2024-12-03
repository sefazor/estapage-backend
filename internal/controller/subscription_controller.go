package controller

import (
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stripe/stripe-go/v74"
	"github.com/stripe/stripe-go/v74/customer"
	"github.com/stripe/stripe-go/v74/subscription"
	"github.com/stripe/stripe-go/v74/webhook"

	"estepage_backend/internal/model"
	"estepage_backend/pkg/database"
	"estepage_backend/pkg/email"
	"estepage_backend/pkg/utils/jwt"
)

type SubscriptionInput struct {
	PlanID string `json:"plan_id" validate:"required"`
}

func InitSubscriptionController() {}

func ListPlans(c *fiber.Ctx) error {
	var plans []model.Subscription
	if err := database.DB.Find(&plans).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not fetch subscription plans",
		})
	}

	return c.JSON(plans)
}

func Subscribe(c *fiber.Ctx) error {
	input := new(SubscriptionInput)
	if err := c.BodyParser(input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid input",
		})
	}

	claims := c.Locals("user").(*jwt.Claims)

	var plan model.Subscription
	if err := database.DB.First(&plan, "stripe_price_id = ?", input.PlanID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Subscription plan not found",
		})
	}

	var user model.User
	if err := database.DB.First(&user, claims.UserID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	stripe.Key = os.Getenv("STRIPE_SECRET_KEY")

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

	expiresAt := time.Unix(stripeSubscription.CurrentPeriodEnd, 0).Format(time.RFC3339)

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

	if email.GlobalEmailService != nil {
		expiresAt, _ := time.Parse(time.RFC3339, userSubscription.ExpiresAt)
		err := email.GlobalEmailService.SendSubscriptionStartedEmail(
			user.Email,
			user.CompanyName,
			plan.Name,
			plan.Duration,
			plan.Price,
			"USD",
			plan.MaxListings,
			expiresAt,
			false,
		)
		if err != nil {
			log.Printf("Could not send subscription email: %v", err)
		}
	}

	return c.JSON(fiber.Map{
		"message":      "Subscription created successfully",
		"subscription": userSubscription,
	})
}

func CancelSubscription(c *fiber.Ctx) error {
	claims := c.Locals("user").(*jwt.Claims)

	var userSub model.UserSubscription
	if err := database.DB.Where("user_id = ? AND status = ?", claims.UserID, "active").
		Preload("User").
		Preload("Subscription").
		First(&userSub).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "No active subscription found",
		})
	}

	stripe.Key = os.Getenv("STRIPE_SECRET_KEY")
	_, err := subscription.Cancel(userSub.StripeSubID, nil)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not cancel Stripe subscription",
		})
	}

	userSub.Status = "cancelled"
	if err := database.DB.Save(&userSub).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not update subscription status",
		})
	}

	if email.GlobalEmailService != nil {
		err := email.GlobalEmailService.SendSubscriptionCancelledEmail(
			userSub.User.Email,
			userSub.User.CompanyName,
			userSub.Subscription.Name,
			time.Now().Add(24*time.Hour),
		)
		if err != nil {
			log.Printf("Could not send subscription cancellation email: %v", err)
		}
	}

	return c.JSON(fiber.Map{
		"message": "Subscription cancelled successfully",
	})
}

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

	log.Printf("Processing Stripe webhook event: %s", event.Type)

	switch event.Type {
	case "customer.subscription.deleted":
		var subData struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(event.Data.Raw, &subData); err != nil {
			return c.Status(fiber.StatusBadRequest).Send(nil)
		}

		log.Printf("Processing subscription deletion: %s", subData.ID)

		var userSub model.UserSubscription
		if err := database.DB.Where("stripe_sub_id = ?", subData.ID).
			Preload("User").
			Preload("Subscription").
			First(&userSub).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Could not find subscription",
			})
		}

		if err := database.DB.Model(&userSub).Update("status", "cancelled").Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Could not update subscription status",
			})
		}

		log.Printf("Subscription %s cancelled successfully", subData.ID)

	case "customer.subscription.renewed":
		var subData struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(event.Data.Raw, &subData); err != nil {
			return c.Status(fiber.StatusBadRequest).Send(nil)
		}

		log.Printf("Processing subscription renewal: %s", subData.ID)

		var userSub model.UserSubscription
		if err := database.DB.Where("stripe_sub_id = ?", subData.ID).
			Preload("User").
			Preload("Subscription").
			First(&userSub).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Could not find subscription",
			})
		}

		if email.GlobalEmailService != nil {
			expiresAt, _ := time.Parse(time.RFC3339, userSub.ExpiresAt)
			err := email.GlobalEmailService.SendSubscriptionStartedEmail(
				userSub.User.Email,
				userSub.User.CompanyName,
				userSub.Subscription.Name,
				userSub.Subscription.Duration,
				userSub.Subscription.Price,
				"USD",
				userSub.Subscription.MaxListings,
				expiresAt,
				true,
			)
			if err != nil {
				log.Printf("Could not send subscription renewal email: %v", err)
			}
		}

		log.Printf("Subscription %s renewed successfully", subData.ID)

	case "customer.subscription.updated":
		var subData struct {
			ID               string `json:"id"`
			CurrentPeriodEnd int64  `json:"current_period_end"`
		}
		if err := json.Unmarshal(event.Data.Raw, &subData); err != nil {
			return c.Status(fiber.StatusBadRequest).Send(nil)
		}

		log.Printf("Processing subscription update: %s", subData.ID)

		expiresAt := time.Unix(subData.CurrentPeriodEnd, 0).Format(time.RFC3339)

		if err := database.DB.Model(&model.UserSubscription{}).
			Where("stripe_sub_id = ?", subData.ID).
			Update("expires_at", expiresAt).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Could not update subscription expiry",
			})
		}

		log.Printf("Subscription %s updated successfully", subData.ID)
	}

	return c.SendStatus(fiber.StatusOK)
}
