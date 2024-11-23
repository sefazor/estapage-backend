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
	"estepage_backend/pkg/utils/location"
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
	// Public routes
	api.Post("/properties/:property_id/leads", controller.CreateLead) // Bu route korumasız olmalı

	// Public Newsletter Routes
	publicNewsletter := api.Group("/newsletter")
	publicNewsletter.Post("/subscribe", controller.AddSubscriber) // Abone olma (public)

	// Protected Newsletter Routes
	protectedNewsletter := api.Group("/newsletter", middleware.AuthMiddleware())
	protectedNewsletter.Get("/subscribers", controller.GetSubscribers) // Aboneleri listeleme (protected)

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

	// Dashboard routes (Protected)
	dashboard := api.Group("/dashboard", middleware.AuthMiddleware())
	dashboard.Get("/stats", controller.GetDashboardStats)

	// Property view recording
	api.Post("/properties/:id/view", controller.RecordPropertyView)

	// Settings routes (Protected)
	settings := api.Group("/settings", middleware.AuthMiddleware())
	settings.Get("/profile", controller.GetProfile)
	settings.Put("/profile", controller.UpdateProfile)

	// Image upload routes (Protected)
	properties.Post("/:property_id/images", controller.UploadPropertyImage)
	properties.Delete("/images/:image_id", controller.DeletePropertyImage)

	// Protected lead routes
	leads := protected.Group("/leads")
	leads.Get("/", controller.GetMyLeads)
	leads.Put("/:id/status", controller.UpdateLeadStatus)
	leads.Put("/:id/read", controller.MarkLeadAsRead)

	// Location routes
	api.Get("/locations/countries", controller.GetLocationData)
	api.Get("/locations/states/:countryCode", controller.GetStatesByCountry)
	api.Get("/locations/cities/:stateCode", controller.GetCitiesByState)

}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	if err := location.Init(); err != nil {
		log.Fatal("Could not initialize location data:", err)
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
		&model.PropertyView{},  // Yeni eklendi
		&model.PropertyStats{}, // Yeni eklendi
		&model.Lead{},
		&model.Subscriber{},
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
