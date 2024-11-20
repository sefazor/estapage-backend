package controller

import (
	"estepage_backend/internal/model"
	"estepage_backend/pkg/database"
	"estepage_backend/pkg/utils/jwt"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

// UploadPropertyImage emlak ilanı için resim yükler
func UploadPropertyImage(c *fiber.Ctx) error {
	claims := c.Locals("user").(*jwt.Claims)

	// Property ID'yi al ve dönüştür
	propertyIDStr := c.Params("property_id")
	propertyID, err := strconv.ParseUint(propertyIDStr, 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid property ID",
		})
	}

	// İlanın varlığını ve sahipliğini kontrol et
	var property model.Property
	if err := database.GetDB().First(&property, propertyID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Property not found",
		})
	}

	// İlan sahibi kontrolü
	if property.UserID != claims.UserID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Not authorized to upload images for this property",
		})
	}

	// Dosya yükleme limiti kontrolü
	var imageCount int64
	database.GetDB().Model(&model.PropertyImage{}).
		Where("property_id = ?", propertyID).
		Count(&imageCount)

	if imageCount >= 16 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Maximum image limit reached (16)",
		})
	}

	// Dosyayı al
	file, err := c.FormFile("image")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "No file uploaded",
		})
	}

	// Dosya tipini kontrol et
	contentType := file.Header.Get("Content-Type")
	if contentType != "image/jpeg" && contentType != "image/png" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Only JPEG and PNG files are allowed",
		})
	}

	// Dosya boyutunu kontrol et (5MB)
	if file.Size > 5*1024*1024 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "File size too large. Maximum size is 5MB",
		})
	}

	// Şimdilik dosyayı temp klasörüne kaydet
	// NOT: Daha sonra S3 entegrasyonu yapılacak
	tempPath := fmt.Sprintf("./tmp/uploads/%d_%s", propertyID, file.Filename)
	if err := c.SaveFile(file, tempPath); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not save file",
		})
	}

	// Veritabanına kaydet
	image := model.PropertyImage{
		PropertyID: uint(propertyID),
		URL:        tempPath,
		Order:      int(imageCount), // Sıradaki index
		IsCover:    imageCount == 0, // İlk resim kapak resmi olsun
	}

	if err := database.GetDB().Create(&image).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not save image record",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Image uploaded successfully",
		"image":   image,
	})
}

// DeletePropertyImage emlak ilanı resmini siler
func DeletePropertyImage(c *fiber.Ctx) error {
	claims := c.Locals("user").(*jwt.Claims)

	imageIDStr := c.Params("image_id")
	imageID, err := strconv.ParseUint(imageIDStr, 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid image ID",
		})
	}

	var image model.PropertyImage
	if err := database.GetDB().Preload("Property").First(&image, imageID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Image not found",
		})
	}

	// İlan sahibi kontrolü
	if image.Property.UserID != claims.UserID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Not authorized to delete this image",
		})
	}

	// Dosyayı sil
	// NOT: S3 entegrasyonunda burayı güncelleyeceğiz
	if err := os.Remove(image.URL); err != nil {
		log.Printf("Could not delete file: %v", err)
	}

	// Veritabanından sil
	if err := database.GetDB().Delete(&image).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not delete image",
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}
