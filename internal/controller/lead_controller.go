package controller

import (
	"estepage_backend/internal/model"
	"estepage_backend/pkg/database"
	"estepage_backend/pkg/email"
	"estepage_backend/pkg/utils/jwt"
	"log"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

type LeadInput struct {
	Name    string `json:"name" validate:"required"`
	Email   string `json:"email" validate:"required,email"`
	Phone   string `json:"phone" validate:"required"`
	Message string `json:"message"`
}

func InitLeadController() {}

func CreatePropertyLead(c *fiber.Ctx) error {
	propertyIDStr := c.Params("property_id")
	propertyID, err := strconv.ParseUint(propertyIDStr, 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid property ID",
		})
	}

	var property model.Property
	if err := database.GetDB().Preload("Images").Preload("User").First(&property, propertyID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Property not found",
		})
	}

	input := new(LeadInput)
	if err := c.BodyParser(input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid input",
		})
	}

	// Property detaylarını pointer olarak set et
	title := property.Title
	price := property.Price
	currency := string(property.Currency)
	var image string
	if len(property.Images) > 0 {
		image = property.Images[0].URL
	}

	lead := model.Lead{
		PropertyID:       uint(propertyID),
		UserID:           property.UserID,
		Name:             input.Name,
		Email:            input.Email,
		Phone:            input.Phone,
		Message:          input.Message,
		Status:           "new",
		Source:           model.LeadSourceProperty,
		PropertyTitle:    &title,
		PropertyPrice:    &price,
		PropertyImage:    &image,
		PropertyCurrency: &currency,
	}

	if err := database.GetDB().Create(&lead).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not create lead",
		})
	}

	if email.GlobalEmailService != nil {
		err := email.GlobalEmailService.SendLeadNotificationEmail(
			property.User.Email,
			property.Title,
			input.Name,
			input.Email,
			input.Phone,
			input.Message,
		)
		if err != nil {
			log.Printf("Could not send lead notification email: %v", err)
		}
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Your inquiry has been sent successfully. The agent will contact you soon.",
	})
}

func CreateProfileLead(c *fiber.Ctx) error {
	userIDStr := c.Params("user_id")
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid user ID",
		})
	}

	var user model.User
	if err := database.GetDB().First(&user, userID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "User not found",
		})
	}

	input := new(LeadInput)
	if err := c.BodyParser(input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid input",
		})
	}

	lead := model.Lead{
		UserID:  uint(userID),
		Name:    input.Name,
		Email:   input.Email,
		Phone:   input.Phone,
		Message: input.Message,
		Status:  "new",
		Source:  model.LeadSourceProfile,
	}

	if err := database.GetDB().Create(&lead).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not create lead",
		})
	}

	if email.GlobalEmailService != nil {
		err := email.GlobalEmailService.SendLeadNotificationEmail(
			user.Email,
			"Profile Lead",
			input.Name,
			input.Email,
			input.Phone,
			input.Message,
		)
		if err != nil {
			log.Printf("Could not send lead notification email: %v", err)
		}
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Your message has been sent successfully. The agent will contact you soon.",
	})
}

func GetMyLeads(c *fiber.Ctx) error {
	claims := c.Locals("user").(*jwt.Claims)

	var leads []model.Lead
	query := database.GetDB().Where("user_id = ?", claims.UserID)

	// Filtreler
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}

	if readStatus := c.Query("read"); readStatus != "" {
		query = query.Where("read_status = ?", readStatus == "true")
	}

	if propertyID := c.Query("property_id"); propertyID != "" {
		query = query.Where("property_id = ?", propertyID)
	}

	if source := c.Query("source"); source != "" {
		query = query.Where("source = ?", source)
	}

	// Sıralama
	if sortBy := c.Query("sort"); sortBy != "" {
		query = query.Order(sortBy)
	} else {
		query = query.Order("created_at desc")
	}

	if err := query.Find(&leads).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not fetch leads",
		})
	}

	return c.JSON(fiber.Map{
		"leads": leads,
		"total": len(leads),
	})
}

func UpdateLeadStatus(c *fiber.Ctx) error {
	claims := c.Locals("user").(*jwt.Claims)
	leadID := c.Params("id")

	var lead model.Lead
	if err := database.GetDB().First(&lead, leadID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Lead not found",
		})
	}

	if lead.UserID != claims.UserID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Not authorized to update this lead",
		})
	}

	input := struct {
		Status string `json:"status"`
	}{}

	if err := c.BodyParser(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid input",
		})
	}

	// Status kontrolü
	switch model.LeadStatus(input.Status) {
	case model.LeadStatusNew,
		model.LeadStatusRead,
		model.LeadStatusContacted,
		model.LeadStatusNoResponse,
		model.LeadStatusCompleted:
		// Geçerli status
	default:
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid status value",
			"valid_statuses": []string{
				string(model.LeadStatusNew),
				string(model.LeadStatusRead),
				string(model.LeadStatusContacted),
				string(model.LeadStatusNoResponse),
				string(model.LeadStatusCompleted),
			},
		})
	}

	if err := database.GetDB().Model(&lead).Update("status", input.Status).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not update lead status",
		})
	}

	return c.JSON(fiber.Map{
		"message": "Lead status updated successfully",
		"lead":    lead,
	})
}

func MarkLeadAsRead(c *fiber.Ctx) error {
	claims := c.Locals("user").(*jwt.Claims)
	leadID := c.Params("id")

	var lead model.Lead
	if err := database.GetDB().First(&lead, leadID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Lead not found",
		})
	}

	if lead.UserID != claims.UserID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Not authorized to update this lead",
		})
	}

	if err := database.GetDB().Model(&lead).Update("read_status", true).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not mark lead as read",
		})
	}

	return c.SendStatus(fiber.StatusOK)
}
