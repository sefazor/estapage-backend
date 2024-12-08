package controller

import (
	"estepage_backend/internal/model"
	"estepage_backend/pkg/database"
	"estepage_backend/pkg/utils/jwt"
	"net/mail"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

type NewsletterSubscriptionInput struct {
	Email string `json:"email" validate:"required,email"`
}

// PublicSubscribe emlakçının sayfasından bültene abone olma
func PublicSubscribe(c *fiber.Ctx) error {
	// URL'den emlakçı ID'sini al
	userIDStr := c.Params("user_id")
	if userIDStr == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Agent ID is required",
		})
	}

	// String'i uint'e çevir
	userID, err := strconv.ParseUint(userIDStr, 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid agent ID format",
		})
	}

	// Input kontrolü
	var input NewsletterSubscriptionInput
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid input format",
		})
	}

	// Email formatı kontrolü
	if _, err := mail.ParseAddress(input.Email); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid email format",
		})
	}

	// Aynı email için kontrol
	var existingSubscriber model.Subscriber
	if err := database.GetDB().Where("user_id = ? AND email = ?", userID, input.Email).
		First(&existingSubscriber).Error; err == nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "Already subscribed to this agent's newsletter",
		})
	}

	// Yeni abone kaydı
	subscriber := model.Subscriber{
		UserID: uint(userID),
		Email:  input.Email,
	}

	if err := database.GetDB().Create(&subscriber).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not complete subscription",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Successfully subscribed to newsletter",
	})
}

// GetMySubscribers emlakçının kendi abonelerini görmesi
func GetMySubscribers(c *fiber.Ctx) error {
	claims := c.Locals("user").(*jwt.Claims)

	var subscribers []model.Subscriber
	if err := database.GetDB().Where("user_id = ?", claims.UserID).Find(&subscribers).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not fetch subscribers",
		})
	}

	return c.JSON(fiber.Map{
		"total_subscribers": len(subscribers),
		"subscribers":       subscribers,
	})
}
