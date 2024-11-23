package controller

import (
	"estepage_backend/internal/model"
	"estepage_backend/pkg/database"
	"estepage_backend/pkg/utils/jwt"
	"fmt"
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

const MaxPropertyImages = 16

type PropertyInput struct {
	Title       string               `json:"title" validate:"required"`
	Type        model.PropertyType   `json:"type" validate:"required"`
	Status      model.PropertyStatus `json:"status" validate:"required"`
	Price       float64              `json:"price" validate:"required"`
	Currency    model.Currency       `json:"currency" validate:"required"`
	Description string               `json:"description" validate:"required"`

	// Location fields
	CountryCode string `json:"country_code" validate:"required"`
	CountryName string `json:"country_name" validate:"required"`
	StateCode   string `json:"state_code" validate:"required"`
	StateName   string `json:"state_name" validate:"required"`
	City        string `json:"city" validate:"required"`
	District    string `json:"district"`
	FullAddress string `json:"full_address" validate:"required"`

	// Features fields
	Bedrooms        int  `json:"bedrooms" binding:"required,min=0"` // Minimum 0, zorunlu
	Bathrooms       int  `json:"bathrooms" binding:"required,min=0"`
	GarageSpaces    int  `json:"garage_spaces" binding:"omitempty,min=0"`
	AreaSqFt        int  `json:"area_sq_ft" binding:"omitempty,min=0"`
	YearBuilt       int  `json:"year_built" binding:"omitempty,min=0"`
	SwimmingPool    bool `json:"swimming_pool"`
	Garden          bool `json:"garden"`
	AirConditioning bool `json:"air_conditioning"`
	CentralHeating  bool `json:"central_heating"`
	SecuritySystem  bool `json:"security_system"`

	Images []string `json:"images"`
}

// CreateProperty yeni emlak ilanı oluşturur
func CreateProperty(c *fiber.Ctx) error {
	claims := c.Locals("user").(*jwt.Claims)
	input := new(PropertyInput)

	if err := c.BodyParser(input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid input",
		})
	}

	// Resim sayısı kontrolü
	if len(input.Images) > MaxPropertyImages {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("Maximum %d images allowed", MaxPropertyImages),
		})
	}

	property := model.Property{
		UserID:          claims.UserID,
		Title:           input.Title,
		Description:     input.Description,
		Price:           input.Price,
		Type:            input.Type,
		Status:          input.Status,
		Currency:        input.Currency,
		CountryCode:     input.CountryCode,
		CountryName:     input.CountryName,
		StateCode:       input.StateCode,
		StateName:       input.StateName,
		City:            input.City,
		District:        input.District,
		FullAddress:     input.FullAddress,
		Bedrooms:        input.Bedrooms,
		Bathrooms:       input.Bathrooms,
		GarageSpaces:    input.GarageSpaces,
		AreaSqFt:        input.AreaSqFt,
		YearBuilt:       input.YearBuilt,
		SwimmingPool:    input.SwimmingPool,
		Garden:          input.Garden,
		AirConditioning: input.AirConditioning,
		CentralHeating:  input.CentralHeating,
		SecuritySystem:  input.SecuritySystem,
	}

	tx := database.GetDB().Begin()

	if err := tx.Create(&property).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not create property",
		})
	}

	for i, imageURL := range input.Images {
		if strings.HasPrefix(imageURL, "https://"+os.Getenv("AWS_BUCKET_NAME")) {
			image := model.PropertyImage{
				PropertyID: property.ID,
				URL:        imageURL,
				Order:      i,
				IsCover:    i == 0,
			}
			if err := tx.Create(&image).Error; err != nil {
				tx.Rollback()
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": "Could not save images",
				})
			}
		}
	}

	if err := tx.Commit().Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not complete the property creation",
		})
	}

	// Property'yi ilişkileriyle birlikte yükle
	database.GetDB().Preload("Images", func(db *gorm.DB) *gorm.DB {
		return db.Order("property_images.order ASC")
	}).First(&property, property.ID)

	return c.Status(fiber.StatusCreated).JSON(property)
}

// UpdateProperty emlak ilanını günceller
func UpdateProperty(c *fiber.Ctx) error {
	claims := c.Locals("user").(*jwt.Claims)
	id := c.Params("id")
	input := new(PropertyInput)

	if err := c.BodyParser(input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid input",
		})
	}

	// Resim sayısı kontrolü
	if len(input.Images) > MaxPropertyImages {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("Maximum %d images allowed", MaxPropertyImages),
		})
	}

	var property model.Property
	if err := database.GetDB().First(&property, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Property not found",
		})
	}

	// Yetki kontrolü
	if property.UserID != claims.UserID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Not authorized to update this property",
		})
	}

	tx := database.GetDB().Begin()

	// Property bilgilerini güncelle
	property.Title = input.Title
	property.Type = input.Type
	property.Status = input.Status
	property.Price = input.Price
	property.Currency = input.Currency
	property.Description = input.Description
	property.CountryCode = input.CountryCode
	property.CountryName = input.CountryName
	property.StateCode = input.StateCode
	property.StateName = input.StateName
	property.City = input.City
	property.District = input.District
	property.FullAddress = input.FullAddress

	if err := tx.Save(&property).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not update property",
		})
	}

	// Mevcut resimleri sil
	if err := tx.Where("property_id = ?", property.ID).Delete(&model.PropertyImage{}).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not update images",
		})
	}

	// Yeni resimleri kaydet
	for i, imageURL := range input.Images {
		image := model.PropertyImage{
			PropertyID: property.ID,
			URL:        imageURL,
			Order:      i,
			IsCover:    i == 0,
		}
		if err := tx.Create(&image).Error; err != nil {
			tx.Rollback()
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Could not save new images",
			})
		}
	}

	if err := tx.Commit().Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not complete the update",
		})
	}

	// Güncellenmiş property'yi ilişkileriyle birlikte yükle
	database.GetDB().Preload("Images", func(db *gorm.DB) *gorm.DB {
		return db.Order("property_images.order ASC")
	}).First(&property, property.ID)

	return c.JSON(property)
}

// ListUserProperties belirli bir kullanıcının public ilanlarını listeler
func ListUserProperties(c *fiber.Ctx) error {
	username := c.Params("username")

	var user model.User
	if err := database.GetDB().Where("username = ?", username).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "User not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not fetch user",
		})
	}

	var properties []model.Property
	if err := database.GetDB().Where("user_id = ? AND status = ?", user.ID, "active").
		Preload("Images", func(db *gorm.DB) *gorm.DB {
			return db.Order("property_images.order ASC")
		}).
		Order("created_at desc").
		Find(&properties).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not fetch properties",
		})
	}

	return c.JSON(fiber.Map{
		"user": fiber.Map{
			"username":     user.Username,
			"company_name": user.CompanyName,
		},
		"properties": properties,
	})
}

// GetPropertyBySlug ilan detayını URL'den alır
func GetPropertyBySlug(c *fiber.Ctx) error {
	username := c.Params("username")
	propertySlug := c.Params("property_slug")

	var user model.User
	if err := database.GetDB().Where("username = ?", username).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "User not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not fetch user",
		})
	}

	var property model.Property
	if err := database.GetDB().Where("user_id = ? AND status = ? AND slug = ?",
		user.ID, "active", propertySlug).
		Preload("Images", func(db *gorm.DB) *gorm.DB {
			return db.Order("property_images.order ASC")
		}).
		First(&property).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Property not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not fetch property",
		})
	}

	return c.JSON(fiber.Map{
		"user": fiber.Map{
			"username":     user.Username,
			"company_name": user.CompanyName,
		},
		"property": property,
	})
}

// ListMyProperties kullanıcının kendi ilanlarını listeler
func ListMyProperties(c *fiber.Ctx) error {
	claims := c.Locals("user").(*jwt.Claims)

	var properties []model.Property
	if err := database.GetDB().Where("user_id = ?", claims.UserID).
		Preload("Images", func(db *gorm.DB) *gorm.DB {
			return db.Order("property_images.order ASC")
		}).
		Order("created_at desc").
		Find(&properties).Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not fetch properties",
		})
	}

	return c.JSON(properties)
}

// DeleteProperty emlak ilanını siler
func DeleteProperty(c *fiber.Ctx) error {
	claims := c.Locals("user").(*jwt.Claims)
	id := c.Params("id")

	var property model.Property // property.Property yerine model.Property
	if err := database.GetDB().First(&property, id).Error; err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Property not found",
		})
	}

	// Yetki kontrolü
	if property.UserID != claims.UserID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Not authorized to delete this property",
		})
	}

	tx := database.GetDB().Begin()

	// Property'yi ve ilişkili kayıtları sil
	if err := tx.Delete(&property).Error; err != nil {
		tx.Rollback()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not delete property",
		})
	}

	if err := tx.Commit().Error; err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Could not complete deletion",
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}
