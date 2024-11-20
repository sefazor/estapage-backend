package controller

import (
	"estepage_backend/internal/model"
	"estepage_backend/pkg/database"
	"estepage_backend/pkg/utils/jwt"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
)

type LeadInput struct {
	Name    string `json:"name" validate:"required"`
	Email   string `json:"email" validate:"required,email"`
	Phone   string `json:"phone" validate:"required"`
	Message string `json:"message"`
}

func CreateLead(c *fiber.Ctx) error {
	propertyIDStr := c.Params("property_id")
	propertyID, err := strconv.ParseUint(propertyIDStr, 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid property ID",
		})
	}

	// Property'nin varlığını kontrol et
	var property model.Property
	if err := database.GetDB().First(&property, propertyID).Error; err != nil {
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

	lead := model.Lead{
		PropertyID: uint(propertyID),
		Name:       input.Name,
		Email:      input.Email,
		Phone:      input.Phone,
		Message:    input.Message,
		Status:     "new",
	}

	if err := database.GetDB().Create(&lead).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not create lead",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Your inquiry has been sent successfully. The agent will contact you soon.",
	})
}

// GetMyLeads kullanıcının tüm ilanlarına gelen başvuruları listeler
func GetMyLeads(c *fiber.Ctx) error {
	claims := c.Locals("user").(*jwt.Claims)

	var leads []model.Lead
	query := database.GetDB().
		Joins("JOIN properties ON leads.property_id = properties.id").
		Where("properties.user_id = ?", claims.UserID).
		Preload("Property")

	// Filtreleme
	if status := c.Query("status"); status != "" {
		query = query.Where("leads.status = ?", status)
	}

	if readStatus := c.Query("read"); readStatus != "" {
		query = query.Where("leads.read_status = ?", readStatus == "true")
	}

	if propertyID := c.Query("property_id"); propertyID != "" {
		query = query.Where("leads.property_id = ?", propertyID)
	}

	// Sıralama
	if sortBy := c.Query("sort"); sortBy != "" {
		query = query.Order(sortBy)
	} else {
		query = query.Order("leads.created_at desc")
	}

	if err := query.Find(&leads).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not fetch leads",
		})
	}

	return c.JSON(leads)
}

// UpdateLeadStatus lead durumunu günceller
func UpdateLeadStatus(c *fiber.Ctx) error {
	claims := c.Locals("user").(*jwt.Claims)
	leadID := c.Params("id")

	var lead model.Lead
	if err := database.GetDB().Preload("Property").First(&lead, leadID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Lead not found",
		})
	}

	// İlan sahibi kontrolü
	if lead.Property.UserID != claims.UserID {
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

	// Status değerini kontrol et
	validStatuses := map[string]bool{
		"new":       true,
		"contacted": true,
		"qualified": true,
		"converted": true,
		"closed":    true,
	}

	if !validStatuses[input.Status] {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid status value",
		})
	}

	updates := map[string]interface{}{
		"status": input.Status,
	}

	// Eğer durum "contacted" ise contacted_at'i güncelle
	if input.Status == "contacted" {
		now := time.Now()
		updates["contacted_at"] = &now
	}

	if err := database.GetDB().Model(&lead).Updates(updates).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not update lead status",
		})
	}

	return c.JSON(lead)
}

// MarkLeadAsRead lead'i okundu olarak işaretler
func MarkLeadAsRead(c *fiber.Ctx) error {
	claims := c.Locals("user").(*jwt.Claims)
	leadID := c.Params("id")

	var lead model.Lead
	if err := database.GetDB().Preload("Property").First(&lead, leadID).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Lead not found",
		})
	}

	// İlan sahibi kontrolü
	if lead.Property.UserID != claims.UserID {
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
