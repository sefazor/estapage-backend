package controller

import (
	"crypto/rand"
	"encoding/hex"
	"estepage_backend/internal/model"
	"estepage_backend/pkg/database"
	"estepage_backend/pkg/email"
	"estepage_backend/pkg/utils/jwt"
	"log"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type RegisterInput struct {
	Email       string `json:"email" validate:"required,email"`
	Password    string `json:"password" validate:"required,min=6"`
	CompanyName string `json:"company_name" validate:"required"`
}

type LoginInput struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type RequestPasswordResetInput struct {
	Email string `json:"email" validate:"required,email"`
}

type ResetPasswordInput struct {
	Token    string `json:"token" validate:"required"`
	Password string `json:"password" validate:"required,min=6"`
}

func InitAuthController() {}

func generateUsername(companyName string) string {
	username := strings.ToLower(companyName)
	username = strings.ReplaceAll(username, " ", "-")
	username = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			return r
		}
		return -1
	}, username)
	return username
}

func Register(c *fiber.Ctx) error {
	input := new(RegisterInput)
	if err := c.BodyParser(input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid input",
		})
	}

	var existingUser model.User
	if err := database.GetDB().Where("email = ?", input.Email).First(&existingUser).Error; err == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Email already exists",
		})
	}

	username := generateUsername(input.CompanyName)

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not hash password",
		})
	}

	user := model.User{
		Email:       input.Email,
		Password:    string(hashedPassword),
		Username:    username,
		CompanyName: input.CompanyName,
	}

	if err := database.GetDB().Create(&user).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not create user",
		})
	}

	token, err := jwt.GenerateToken(user.ID, user.Email, user.CompanyName)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not generate token",
		})
	}

	if email.GlobalEmailService != nil {
		if err := email.GlobalEmailService.SendWelcomeEmail(user.Email, input.CompanyName); err != nil {
			log.Printf("Could not send welcome email: %v", err)
		}
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Registration successful",
		"token":   token,
		"user":    user.GetPublicProfile(),
	})
}

func Login(c *fiber.Ctx) error {
	input := new(LoginInput)
	if err := c.BodyParser(input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid input",
		})
	}

	var user model.User
	if err := database.GetDB().Where("email = ?", input.Email).First(&user).Error; err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid credentials",
		})
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Invalid credentials",
		})
	}

	token, err := jwt.GenerateToken(user.ID, user.Email, user.CompanyName)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not generate token",
		})
	}

	return c.JSON(fiber.Map{
		"token": token,
		"user": fiber.Map{
			"id":           user.ID,
			"email":        user.Email,
			"company_name": user.CompanyName,
		},
	})
}

func GetMe(c *fiber.Ctx) error {
	claims := c.Locals("user").(*jwt.Claims)

	var user model.User
	if err := database.GetDB().First(&user, claims.UserID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "User not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not fetch user",
		})
	}

	return c.JSON(fiber.Map{
		"user": fiber.Map{
			"id":           user.ID,
			"email":        user.Email,
			"username":     user.Username,
			"company_name": user.CompanyName,
			"created_at":   user.CreatedAt,
		},
	})
}

func generateResetToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func RequestPasswordReset(c *fiber.Ctx) error {
	input := new(RequestPasswordResetInput)
	if err := c.BodyParser(input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid input",
		})
	}

	var user model.User
	if err := database.GetDB().Where("email = ?", input.Email).First(&user).Error; err != nil {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"message": "If your email exists in our system, you will receive a password reset link",
		})
	}

	token, err := generateResetToken()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not generate reset token",
		})
	}

	expires := time.Now().Add(1 * time.Hour)
	if err := database.GetDB().Model(&user).Updates(map[string]interface{}{
		"password_reset_token": token,
		"reset_token_expires":  expires,
	}).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not save reset token",
		})
	}

	if email.GlobalEmailService != nil {
		if err := email.GlobalEmailService.SendPasswordResetEmail(user.Email, token); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Could not send reset email",
			})
		}
	}

	return c.JSON(fiber.Map{
		"message": "If your email exists in our system, you will receive a password reset link",
	})
}

func ResetPassword(c *fiber.Ctx) error {
	input := new(ResetPasswordInput)
	if err := c.BodyParser(input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid input",
		})
	}

	var user model.User
	if err := database.GetDB().Where("password_reset_token = ?", input.Token).First(&user).Error; err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid or expired reset token",
		})
	}

	if time.Now().After(user.ResetTokenExpires) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Reset token has expired",
		})
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not hash password",
		})
	}

	if err := database.GetDB().Model(&user).Updates(map[string]interface{}{
		"password":             string(hashedPassword),
		"password_reset_token": "",
		"reset_token_expires":  time.Time{},
	}).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not update password",
		})
	}

	if email.GlobalEmailService != nil {
		if err := email.GlobalEmailService.SendPasswordChangedEmail(user.Email); err != nil {
			log.Printf("Could not send password changed notification: %v", err)
		}
	}

	return c.JSON(fiber.Map{
		"message": "Password has been reset successfully",
	})
}
