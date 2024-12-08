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
	portalsession "github.com/stripe/stripe-go/v74/billingportal/session"
	"github.com/stripe/stripe-go/v74/checkout/session"
	stripeinvoice "github.com/stripe/stripe-go/v74/invoice"
	"github.com/stripe/stripe-go/v74/subscription"
	"github.com/stripe/stripe-go/v74/webhook"

	"estepage_backend/internal/model"
	"estepage_backend/pkg/database"
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
		SuccessURL:        stripe.String(os.Getenv("FRONTEND_URL") + "/settings/subscription/success"),
		CancelURL:         stripe.String(os.Getenv("FRONTEND_URL") + "/settings/subscription/cancel"),
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

func GetMySubscription(c *fiber.Ctx) error {
	claims := c.Locals("user").(*jwt.Claims)

	var userSub model.UserSubscription
	if err := database.DB.Where("user_id = ? AND status = ?", claims.UserID, "active").First(&userSub).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "No active subscription found",
		})
	}

	// Stripe'dan güncel subscription bilgisini al
	stripeSub, err := subscription.Get(userSub.StripeSubID, nil)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not fetch subscription details",
		})
	}

	return c.JSON(fiber.Map{
		"subscription": userSub,
		"details": fiber.Map{
			"current_period_end": time.Unix(stripeSub.CurrentPeriodEnd, 0),
			"status":             stripeSub.Status,
		},
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

	// Stripe'dan subscription detaylarını al
	stripeSub, err := subscription.Get(userSub.StripeSubID, nil)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not fetch subscription details from Stripe",
		})
	}

	// İptal parametrelerini ayarla
	params := &stripe.SubscriptionParams{
		CancelAtPeriodEnd: stripe.Bool(true),
	}

	// Stripe'da aboneliği iptal et
	_, err = subscription.Update(userSub.StripeSubID, params)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not cancel subscription with Stripe",
		})
	}

	// Veritabanında durumu güncelle
	userSub.Status = "cancelling"
	userSub.CancellationDate = time.Now()
	if err := database.DB.Save(&userSub).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not update subscription status",
		})
	}

	// Dönem sonu tarihini al
	periodEnd := time.Unix(stripeSub.CurrentPeriodEnd, 0)
	daysRemaining := int(time.Until(periodEnd).Hours() / 24)

	return c.JSON(fiber.Map{
		"message": "Subscription cancelled successfully",
		"details": fiber.Map{
			"status":             "cancelling",
			"current_period_end": periodEnd,
			"days_remaining":     daysRemaining,
			"cancellation_date":  userSub.CancellationDate,
			"plan_access_until": fmt.Sprintf("Plan özelliklerini %s tarihine kadar kullanabilirsiniz",
				periodEnd.Format("2 January 2006")),
		},
	})
}

func CreateCustomerPortalSession(c *fiber.Ctx) error {
	claims := c.Locals("user").(*jwt.Claims)

	var user model.User
	if err := database.DB.First(&user, claims.UserID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	params := &stripe.BillingPortalSessionParams{
		Customer:  stripe.String(user.StripeCustomerID),
		ReturnURL: stripe.String(os.Getenv("FRONTEND_URL") + "/settings/subscription"),
	}

	session, err := portalsession.New(params)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not create portal session",
		})
	}

	return c.JSON(fiber.Map{
		"url": session.URL,
	})
}

func HandleStripeWebhook(c *fiber.Ctx) error {
	webhookSecret := os.Getenv("STRIPE_WEBHOOK_SECRET")
	payload := c.Body()
	signatureHeader := c.Get("Stripe-Signature")

	event, err := webhook.ConstructEvent(payload, signatureHeader, webhookSecret)
	if err != nil {
		log.Printf("Webhook Error: %v", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid webhook signature",
		})
	}

	log.Printf("Handling webhook event: %s", event.Type)

	switch event.Type {
	case "checkout.session.completed":
		var s stripe.CheckoutSession
		if err := json.Unmarshal(event.Data.Raw, &s); err != nil {
			log.Printf("Error parsing webhook JSON: %v", err)
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

	case "customer.subscription.updated":
		var sub stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
			return c.Status(fiber.StatusBadRequest).Send(nil)
		}

		var userSub model.UserSubscription
		if err := database.DB.Where("stripe_sub_id = ?", sub.ID).First(&userSub).Error; err != nil {
			return c.Status(fiber.StatusNotFound).Send(nil)
		}

		userSub.Status = string(sub.Status)
		if err := database.DB.Save(&userSub).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).Send(nil)
		}

	case "customer.subscription.deleted":
		var sub stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
			return c.Status(fiber.StatusBadRequest).Send(nil)
		}

		var userSub model.UserSubscription
		if err := database.DB.Where("stripe_sub_id = ?", sub.ID).First(&userSub).Error; err != nil {
			return c.Status(fiber.StatusNotFound).Send(nil)
		}

		userSub.Status = "cancelled"
		if err := database.DB.Save(&userSub).Error; err != nil {
			return c.Status(fiber.StatusInternalServerError).Send(nil)
		}
	}

	return c.SendStatus(fiber.StatusOK)
}

func GetInvoices(c *fiber.Ctx) error {
	claims := c.Locals("user").(*jwt.Claims)

	var user model.User
	if err := database.GetDB().First(&user, claims.UserID).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not fetch user details",
		})
	}

	if user.StripeCustomerID == "" {
		return c.JSON(fiber.Map{
			"invoices": []interface{}{},
		})
	}

	params := &stripe.InvoiceListParams{
		Customer: stripe.String(user.StripeCustomerID),
	}
	params.Filters.AddFilter("limit", "", "100")

	iterator := stripeinvoice.List(params)

	var invoices []map[string]interface{}
	for iterator.Next() {
		inv := iterator.Invoice()

		invoices = append(invoices, map[string]interface{}{
			"id":       inv.Number,
			"date":     time.Unix(inv.Created, 0).Format("January 2, 2006"),
			"amount":   float64(inv.AmountPaid) / 100,
			"currency": string(inv.Currency),
			"status":   string(inv.Status),
			"pdf":      inv.InvoicePDF,
		})
	}

	if err := iterator.Err(); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not fetch invoices from Stripe",
		})
	}

	return c.JSON(fiber.Map{
		"invoices": invoices,
	})
}

// Success ve Cancel handler'ları
func HandleSubscriptionSuccess(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"message": "Subscription successful",
		"status":  "success",
	})
}

func HandleSubscriptionCancel(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"message": "Subscription cancelled",
		"status":  "cancelled",
	})
}
