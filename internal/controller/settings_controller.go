package controller

import (
	"estepage_backend/internal/model"
	"estepage_backend/pkg/database"
	"estepage_backend/pkg/utils/cloudflare"
	"estepage_backend/pkg/utils/jwt"
	"fmt"
	"log"
	"strings"

	"github.com/gofiber/fiber/v2"
)

type ProfileUpdateInput struct {
	FirstName      string `json:"first_name"`
	LastName       string `json:"last_name"`
	Title          string `json:"title"`
	PhoneNumber    string `json:"phone_number"`
	BusinessEmail  string `json:"business_email"`
	WhatsAppNumber string `json:"whats_app_number"`
	// Avatar field'ı kaldırıldı
	AboutMe      string  `json:"about_me"`
	Experience   int     `json:"experience"`
	TotalClients uint    `json:"total_clients"`
	SoldScore    int     `json:"sold_score"`
	Rating       float64 `json:"rating"`
}

func UpdateProfile(c *fiber.Ctx) error {
	claims := c.Locals("user").(*jwt.Claims)
	input := new(ProfileUpdateInput)

	if err := c.BodyParser(input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid input",
		})
	}

	var user model.User
	if err := database.GetDB().First(&user, claims.UserID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	updates := map[string]interface{}{
		"first_name":       input.FirstName,
		"last_name":        input.LastName,
		"title":            input.Title,
		"phone_number":     input.PhoneNumber,
		"business_email":   input.BusinessEmail,
		"whats_app_number": input.WhatsAppNumber,
		// Avatar güncelleme kaldırıldı
		"experience":    input.Experience,
		"total_clients": input.TotalClients,
		"sold_score":    input.SoldScore,
		"rating":        input.Rating,
		"about_me":      input.AboutMe,
	}

	if err := database.GetDB().Model(&user).Updates(updates).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not update profile",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Profile updated successfully",
		"user":    user.GetPublicProfile(),
	})
}

func GetProfile(c *fiber.Ctx) error {
	claims := c.Locals("user").(*jwt.Claims)

	var user model.User
	if err := database.GetDB().First(&user, claims.UserID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	return c.JSON(user.GetPublicProfile())
}

func UploadAvatar(c *fiber.Ctx) error {
	claims := c.Locals("user").(*jwt.Claims)

	// Kullanıcı bilgilerini al
	var user model.User
	if err := database.GetDB().First(&user, claims.UserID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	// Dosya kontrolü
	file, err := c.FormFile("avatar")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "No avatar image provided",
		})
	}

	// Dosya tipi kontrolü
	if !strings.HasPrefix(file.Header.Get("Content-Type"), "image/") {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "File must be an image",
		})
	}

	// Eğer eski avatar varsa, sil
	if user.Avatar != "" {
		if err := cloudflare.DeleteImage(user.Avatar); err != nil {
			// Hata logla ama işleme devam et
			log.Printf("Error deleting old avatar: %v", err)
		}
	}

	// Yeni avatar'ı yükle
	config := cloudflare.UploadAvatarConfig{
		File:     file,
		Username: user.Username,
	}

	avatarURL, err := cloudflare.UploadAvatar(config)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Could not upload avatar: %v", err),
		})
	}

	// Veritabanını güncelle
	if err := database.GetDB().Model(&user).Update("avatar", avatarURL).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not update avatar",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Avatar uploaded successfully",
		"avatar":  avatarURL,
	})
}
