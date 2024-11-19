package main

import (
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/joho/godotenv"

	"estepage_backend/internal/controller"
	"estepage_backend/internal/middleware"
	"estepage_backend/internal/model"
	"estepage_backend/pkg/database"
	"estepage_backend/pkg/seed"
)

func setupRoutes(app *fiber.App) {
	api := app.Group("/api")

	// Auth Routes
	auth := api.Group("/auth")
	auth.Post("/register", controller.Register)
	auth.Post("/login", controller.Login)

	// Public Properties Routes
	publicProps := api.Group("/p")
	publicProps.Get("/:username", controller.ListUserProperties)               // Kullanıcının tüm ilanları
	publicProps.Get("/:username/:property_slug", controller.GetPropertyBySlug) // İlan detayı

	// Protected Routes (Authentication gerekir)
	protected := api.Group("/", middleware.AuthMiddleware())
	protected.Get("/me", controller.GetMe)

	// Protected Property Routes
	properties := protected.Group("/properties")
	properties.Get("/my", controller.ListMyProperties)   // Kendi ilanlarım
	properties.Post("/", controller.CreateProperty)      // İlan oluştur
	properties.Put("/:id", controller.UpdateProperty)    // İlan güncelle
	properties.Delete("/:id", controller.DeleteProperty) // İlan sil

	// Upload routes
	properties.Post("/:id/images", controller.UploadPropertyImage)

}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	// Database connection
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is not set in .env")
	}

	database.InitDB(dbURL)

	err := database.MigrateDatabase(
		&model.User{},
		&model.Subscription{},
		&model.UserSubscription{},
		&model.Property{},
		&model.PropertyImage{},
	)
	if err != nil {
		log.Printf("Migration warning: %v", err)
	}

	seed.SeedSubscriptionPlans(database.GetDB())

	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": err.Error(),
			})
		},
	})

	app.Use(logger.New())
	app.Use(cors.New())

	setupRoutes(app)

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	log.Printf("Server is running on port %s", port)
	log.Fatal(app.Listen(":" + port))
}
