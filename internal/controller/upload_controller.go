package controller

import (
	"estepage_backend/pkg/utils/jwt"
	"estepage_backend/pkg/utils/storage"

	"github.com/gofiber/fiber/v2"
)

// UploadPropertyImage emlak ilanı için resim yükler
func UploadPropertyImage(c *fiber.Ctx) error {
	claims := c.Locals("user").(*jwt.Claims)
	propertyID := c.Params("property_id")

	// Dosyayı al
	file, err := c.FormFile("image")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "No file uploaded",
		})
	}

	// Resmi yükle
	imageURL, err := storage.UploadImage(file, claims.UserID, propertyID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"url": imageURL,
	})
}
