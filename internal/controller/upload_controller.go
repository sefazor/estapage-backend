package controller

import (
	"estepage_backend/internal/model"
	"estepage_backend/pkg/database"
	"estepage_backend/pkg/utils/cloudflare"
	"estepage_backend/pkg/utils/jwt"
	"fmt"
	"log"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

func UploadPropertyImage(c *fiber.Ctx) error {
	claims := c.Locals("user").(*jwt.Claims)

	propertyID := c.Params("property_id")
	log.Printf("Upload request for propertyID: '%s'", propertyID)

	if propertyID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Property ID is required",
		})
	}

	propertyIDUint, err := strconv.ParseUint(propertyID, 10, 32)
	if err != nil {
		log.Printf("Parse error: %v for value: '%s'", err, propertyID)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid property ID",
		})
	}

	// Property'nin varlığını ve yetkiyi kontrol et
	var property model.Property
	if err := database.GetDB().First(&property, propertyIDUint).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Property not found",
		})
	}

	// Yetki kontrolü
	if property.UserID != claims.UserID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Not authorized",
		})
	}

	// Kaydedilecek görsel sayısını kontrol et
	var imageCount int64
	if err := database.GetDB().Model(&model.PropertyImage{}).
		Where("property_id = ?", propertyIDUint).Count(&imageCount).Error; err != nil {
		log.Printf("Error checking image count: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not check image count",
		})
	}

	if imageCount >= 16 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Maximum image count (16) reached",
		})
	}

	// Dosya kontrolü
	file, err := c.FormFile("image")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "No image provided",
		})
	}

	// Cloudflare'e yükle
	imageURL, err := cloudflare.UploadImage(file)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Could not upload image: %v", err),
		})
	}

	// Veritabanına kaydet
	image := model.PropertyImage{
		PropertyID: uint(propertyIDUint),
		URL:        imageURL,
		Order:      int(imageCount),
		IsCover:    imageCount == 0,
	}

	if err := database.GetDB().Create(&image).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not save image",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(image)
}

func DeletePropertyImage(c *fiber.Ctx) error {
	claims := c.Locals("user").(*jwt.Claims)
	imageID := c.Params("image_id")

	var image model.PropertyImage
	if err := database.GetDB().First(&image, imageID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Image not found",
		})
	}

	// Property'nin sahibi mi kontrol et
	var property model.Property
	if err := database.GetDB().First(&property, image.PropertyID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Property not found",
		})
	}

	if property.UserID != claims.UserID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Not authorized",
		})
	}

	// Cloudflare'den sil
	if image.CloudflareID != "" {
		if err := cloudflare.DeleteImage(image.CloudflareID); err != nil {
			log.Printf("Error deleting image from Cloudflare: %v", err)
		}
	}

	// Database'den sil
	if err := database.GetDB().Delete(&image).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not delete image",
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}
