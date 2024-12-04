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

	// Kullanıcı ve property bilgilerini al
	var user model.User
	if err := database.GetDB().First(&user, claims.UserID).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not fetch user details",
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

	// Upload config'i hazırla ve kullan
	config := cloudflare.UploadImageConfig{
		File:         file,
		Username:     user.Username,
		PropertySlug: property.Slug,
	}

	// Cloudflare R2'ye yükle
	result, err := cloudflare.UploadImage(config) // config'i burada kullan
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Could not upload image: %v", err),
		})
	}

	// Veritabanına kaydet
	image := model.PropertyImage{
		PropertyID:   uint(propertyIDUint),
		URL:          result.URL,
		CloudflareID: result.CloudflareID,
		Order:        int(imageCount),
		IsCover:      imageCount == 0,
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

	// Cloudflare R2'den sil - artık tam URL'i kullanıyoruz
	if err := cloudflare.DeleteImage(image.URL); err != nil {
		log.Printf("Error deleting image from Cloudflare R2: %v", err)
	}

	// Database'den sil
	if err := database.GetDB().Delete(&image).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not delete image",
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}
