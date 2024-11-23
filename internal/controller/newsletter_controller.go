package controller

import (
	"estepage_backend/internal/model"
	"estepage_backend/pkg/database"

	"net/mail"

	"github.com/gofiber/fiber/v2"
)

type SubscriberInput struct {
	UserID uint   `json:"user_id"`
	Email  string `json:"email"`
}

func AddSubscriber(c *fiber.Ctx) error {
	var input SubscriberInput

	// Gelen veriyi parse et
	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid input format"})
	}

	// E-posta formatını kontrol et
	if _, err := mail.ParseAddress(input.Email); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid email format"})
	}

	// Aynı e-posta için daha önce abone kaydı yapılmış mı kontrol et
	var existingSubscriber model.Subscriber
	if err := database.GetDB().Where("user_id = ? AND email = ?", input.UserID, input.Email).First(&existingSubscriber).Error; err == nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "Subscriber already exists"})
	}

	// Yeni aboneyi kaydet
	subscriber := model.Subscriber{
		UserID: input.UserID,
		Email:  input.Email,
	}

	if err := database.GetDB().Create(&subscriber).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to save subscriber"})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"message": "Subscriber added successfully"})
}

func GetSubscribers(c *fiber.Ctx) error {
	// Kullanıcı ID'sini sorgudan al
	userID := c.Query("user_id")
	if userID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "User ID is required",
		})
	}

	var subscribers []model.Subscriber

	// Veritabanından kullanıcıya ait aboneleri getir
	err := database.GetDB().Where("user_id = ?", userID).Find(&subscribers).Error
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to fetch subscribers",
		})
	}

	// Eğer abone yoksa bilgilendirici bir mesaj dön
	if len(subscribers) == 0 {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"message": "No subscribers found for the given user ID",
		})
	}

	// Aboneleri döndür
	return c.Status(fiber.StatusOK).JSON(subscribers)
}
