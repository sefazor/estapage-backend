package controller

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/stripe/stripe-go/v74"
	"github.com/stripe/stripe-go/v74/checkout/session"
	"github.com/stripe/stripe-go/v74/subscription"
	"github.com/stripe/stripe-go/v74/webhook"

	"estepage_backend/internal/model"
	"estepage_backend/pkg/database"
	"estepage_backend/pkg/email"
	"estepage_backend/pkg/utils/jwt"
)

func InitSubscriptionController() {
	stripe.Key = os.Getenv("STRIPE_SECRET_KEY")
}

func ListPlans(c *fiber.Ctx) error {
	var plans []model.Subscription
	if err := database.DB.Find(&plans).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not fetch subscription plans",
		})
	}
	return c.JSON(plans)
}

func CreateCheckoutSession(c *fiber.Ctx) error {
	claims := c.Locals("user").(*jwt.Claims)
	input := new(struct {
		PriceID string `json:"price_id"`
	})

	if err := c.BodyParser(input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid input",
		})
	}

	var user model.User
	if err := database.DB.First(&user, claims.UserID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	params := &stripe.CheckoutSessionParams{
		Mode: stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(input.PriceID),
				Quantity: stripe.Int64(1),
			},
		},
		SuccessURL:        stripe.String(os.Getenv("FRONTEND_URL") + "/payment/success?session_id={CHECKOUT_SESSION_ID}"),
		CancelURL:         stripe.String(os.Getenv("FRONTEND_URL") + "/payment/cancel"),
		CustomerEmail:     stripe.String(user.Email),
		ClientReferenceID: stripe.String(fmt.Sprintf("%d", user.ID)),
	}

	s, err := session.New(params)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Error creating checkout session: %v", err),
		})
	}

	return c.JSON(fiber.Map{
		"url": s.URL,
	})
}

func CancelSubscription(c *fiber.Ctx) error {
	claims := c.Locals("user").(*jwt.Claims)

	var userSub model.UserSubscription
	if err := database.DB.Where("user_id = ? AND status = ?", claims.UserID, "active").First(&userSub).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "No active subscription found",
		})
	}

	// Cancel subscription in Stripe
	_, err := subscription.Cancel(userSub.StripeSubID, nil)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not cancel subscription with Stripe",
		})
	}

	// Update status in database
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

func GetMySubscription(c *fiber.Ctx) error {
	claims := c.Locals("user").(*jwt.Claims)

	var userSub model.UserSubscription
	if err := database.DB.Where("user_id = ? AND status = ?", claims.UserID, "active").First(&userSub).Error; err != nil {
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
	case "checkout.session.completed":
		var s stripe.CheckoutSession
		if err := json.Unmarshal(event.Data.Raw, &s); err != nil {
			return c.Status(fiber.StatusBadRequest).Send(nil)
		}

		userID, _ := strconv.ParseUint(s.ClientReferenceID, 10, 32)

		subscription := model.UserSubscription{
			UserID:       uint(userID),
			StripePlanID: s.Subscription.ID,
			Status:       "active",
			StripeSubID:  s.Subscription.ID,
			ExpiresAt:    time.Now().AddDate(0, 1, 0).Format(time.RFC3339),
		}

		if err := database.DB.Create(&subscription).Error; err != nil {
			log.Printf("Error creating subscription record: %v", err)
			return c.Status(fiber.StatusInternalServerError).Send(nil)
		}

		// Email notification
		var user model.User
		if err := database.DB.First(&user, userID).Error; err == nil && email.GlobalEmailService != nil {
			expiresAt, _ := time.Parse(time.RFC3339, subscription.ExpiresAt)
			err := email.GlobalEmailService.SendSubscriptionStartedEmail(
				user.Email,
				user.CompanyName,
				"Your Plan", // Get plan name from Stripe
				30,          // Standard duration
				0,           // Get price from Stripe
				"USD",
				0, // Get max listings from plan
				expiresAt,
				false,
			)
			if err != nil {
				log.Printf("Could not send subscription email: %v", err)
			}
		}
	}

	return c.SendStatus(fiber.StatusOK)
}
