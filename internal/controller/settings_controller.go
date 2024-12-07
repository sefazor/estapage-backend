package controller

import (
	"encoding/json"
	"estepage_backend/internal/model"
	"estepage_backend/pkg/database"
	"estepage_backend/pkg/email"
	"estepage_backend/pkg/utils/cloudflare"
	"estepage_backend/pkg/utils/jwt"
	"fmt"
	"log"
	"strings"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/datatypes"
)

type ProfileUpdateInput struct {
	Email          string            `json:"email"`
	Password       string            `json:"password"`
	Username       string            `json:"username"`
	CompanyName    string            `json:"company_name"`
	FirstName      string            `json:"first_name"`
	LastName       string            `json:"last_name"`
	Title          string            `json:"title"`
	PhoneNumber    string            `json:"phone_number"`
	BusinessEmail  string            `json:"business_email"`
	WhatsAppNumber string            `json:"whats_app_number"`
	AboutMe        string            `json:"about_me"`
	Experience     int               `json:"experience"`
	TotalClients   uint              `json:"total_clients"`
	SoldScore      int               `json:"sold_score"`
	Rating         float64           `json:"rating"`
	SocialLinks    map[string]string `json:"social_links"`
}

type ChangePasswordInput struct {
	CurrentPassword    string `json:"current_password" validate:"required"`
	NewPassword        string `json:"new_password" validate:"required,min=6"`
	NewPasswordConfirm string `json:"new_password_confirm" validate:"required,eqfield=NewPassword"`
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
	if err := database.DB.First(&user, claims.UserID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	// Hash password if provided
	if input.Password != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Could not hash password",
			})
		}
		user.Password = string(hashedPassword)
	}

	updates := map[string]interface{}{
		"email":            input.Email,
		"username":         input.Username,
		"company_name":     input.CompanyName,
		"first_name":       input.FirstName,
		"last_name":        input.LastName,
		"title":            input.Title,
		"phone_number":     input.PhoneNumber,
		"business_email":   input.BusinessEmail,
		"whats_app_number": input.WhatsAppNumber,
		"experience":       input.Experience,
		"total_clients":    input.TotalClients,
		"sold_score":       input.SoldScore,
		"rating":           input.Rating,
		"about_me":         input.AboutMe,
	}

	// Process social links if provided
	if len(input.SocialLinks) > 0 {
		socialLinksJSON, err := json.Marshal(input.SocialLinks)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Could not process social links",
			})
		}
		updates["social_links"] = datatypes.JSON(socialLinksJSON)
	}

	if err := database.DB.Model(&user).Updates(updates).Error; err != nil {
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

	var user model.User
	if err := database.GetDB().First(&user, claims.UserID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	file, err := c.FormFile("avatar")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "No avatar image provided",
		})
	}

	if !strings.HasPrefix(file.Header.Get("Content-Type"), "image/") {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "File must be an image",
		})
	}

	if user.Avatar != "" {
		if err := cloudflare.DeleteImage(user.Avatar); err != nil {
			log.Printf("Error deleting old avatar: %v", err)
		}
	}

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
func ChangePassword(c *fiber.Ctx) error {
	claims := c.Locals("user").(*jwt.Claims)
	input := new(ChangePasswordInput)

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

	// Mevcut şifreyi kontrol et
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.CurrentPassword)); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Current password is incorrect",
		})
	}

	// Yeni şifreyi hashle
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not hash new password",
		})
	}

	// Şifreyi güncelle
	if err := database.DB.Model(&user).Update("password", string(hashedPassword)).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not update password",
		})
	}

	// Email gönder
	if email.GlobalEmailService != nil {
		err := email.GlobalEmailService.SendPasswordChangedEmail(user.Email)
		if err != nil {
			log.Printf("Error sending password changed email: %v", err)
		}
	}
	return c.JSON(fiber.Map{
		"message": "Password changed successfully",
	})
}
