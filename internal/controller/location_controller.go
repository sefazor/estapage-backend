// internal/controller/location_controller.go
package controller

import (
	"estepage_backend/pkg/utils/location"

	"github.com/gofiber/fiber/v2"
)

func GetLocationData(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"countries": location.GetCountries(),
	})
}

func GetStatesByCountry(c *fiber.Ctx) error {
	countryCode := c.Params("countryCode")
	if countryCode == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Country code is required",
		})
	}

	states := location.GetStatesByCountry(countryCode)
	return c.JSON(fiber.Map{
		"states": states,
	})
}

func GetCitiesByState(c *fiber.Ctx) error {
	stateCode := c.Params("stateCode") // "stateID" yerine "stateCode"
	if stateCode == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "State code is required",
		})
	}

	cities := location.GetCitiesByState(stateCode) // Burayı da string parametre alacak şekilde düzeltelim
	return c.JSON(fiber.Map{
		"cities": cities,
	})
}
