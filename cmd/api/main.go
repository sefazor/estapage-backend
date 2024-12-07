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
	"estepage_backend/pkg/cron"
	"estepage_backend/pkg/database"
	"estepage_backend/pkg/email"
	"estepage_backend/pkg/subscription"
	"estepage_backend/pkg/utils/cloudflare"
	"estepage_backend/pkg/utils/location"
)

func setupRoutes(app *fiber.App) {
	api := app.Group("/api")

	// Auth Routes
	auth := api.Group("/auth")
	auth.Post("/register", controller.Register)
	auth.Post("/login", controller.Login)
	auth.Post("/request-reset", controller.RequestPasswordReset)
	auth.Post("/reset-password", controller.ResetPassword)

	// Public Properties Routes
	publicProps := api.Group("/p")
	publicProps.Get("/:username", controller.ListUserProperties)
	publicProps.Get("/:username/:property_slug", controller.GetPropertyBySlug)

	// Public Newsletter Routes (with subscription check)
	publicNewsletter := api.Group("/newsletter")
	publicNewsletter.Post("/subscribe", middleware.CheckFeatureAccess(subscription.NewsletterForm), controller.AddSubscriber)

	// Protected Newsletter Routes
	protectedNewsletter := api.Group("/newsletter", middleware.AuthMiddleware())
	protectedNewsletter.Get("/subscribers", controller.GetSubscribers)

	// Protected Routes
	protected := api.Group("/", middleware.AuthMiddleware())
	protected.Get("/me", controller.GetMe)

	// Protected Property Routes with subscription checks
	properties := protected.Group("/properties")
	properties.Get("/my", controller.ListMyProperties)
	properties.Post("/", middleware.CheckSubscriptionLimit(), controller.CreateProperty)
	properties.Put("/:id", middleware.CheckPropertyOwnership(), controller.UpdateProperty)
	properties.Delete("/:id", middleware.CheckPropertyOwnership(), controller.DeleteProperty)
	properties.Post("/:property_id/images", middleware.CheckImageLimit(), controller.UploadPropertyImage)
	properties.Delete("/images/:image_id", middleware.CheckPropertyOwnership(), controller.DeletePropertyImage)

	// Lead form with subscription check
	api.Post("/properties/:property_id/leads", middleware.CheckFeatureAccess(subscription.LeadForm), controller.CreateLead)

	// Dashboard routes
	dashboard := api.Group("/dashboard", middleware.AuthMiddleware())
	dashboard.Get("/stats", controller.GetDashboardStats)

	// Property view recording
	api.Post("/properties/:id/view", controller.RecordPropertyView)

	// Settings routes
	settings := api.Group("/settings", middleware.AuthMiddleware())
	settings.Get("/profile", controller.GetProfile)
	settings.Put("/profile", controller.UpdateProfile)
	settings.Post("/avatar", cloudflare.UploadAvatarHandler)

	// Protected lead routes
	leads := protected.Group("/leads")
	leads.Get("/", controller.GetMyLeads)
	leads.Put("/:id/status", controller.UpdateLeadStatus)
	leads.Put("/:id/read", controller.MarkLeadAsRead)

	// Location routes
	api.Get("/locations/countries", controller.GetLocationData)
	api.Get("/locations/states/:countryCode", controller.GetStatesByCountry)
	api.Get("/locations/cities/:stateCode", controller.GetCitiesByState)

	// Subscription routes
	subscriptions := api.Group("/subscriptions")
	subscriptions.Get("/plans", controller.ListPlans)

	subProtected := subscriptions.Use(middleware.AuthMiddleware())
	subProtected.Post("/create-checkout-session", controller.CreateCheckoutSession)
	subProtected.Post("/cancel-subscription", controller.CancelSubscription) // Aktif abonelik iptali
	subProtected.Get("/my", controller.GetMySubscription)

	// Stripe checkout süreç sonuçları
	subscriptions.Get("/payment-success", controller.HandleSubscriptionSuccess)  // Ödeme başarılı
	subscriptions.Get("/payment-cancelled", controller.HandleSubscriptionCancel) // Ödemeden vazgeçildi

	// Stripe webhook
	api.Post("/webhook", controller.HandleStripeWebhook)
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	if err := email.InitEmailService(os.Getenv("RESEND_API_KEY")); err != nil {
		log.Fatal("Could not initialize email service:", err)
	}
	log.Printf("Email service initialized with API key: %s", os.Getenv("RESEND_API_KEY"))

	controller.InitAuthController()
	controller.InitLeadController()
	cron.InitNewsletterCron()
	controller.InitSubscriptionController()
	cron.InitSubscriptionExpiryCron()

	if err := location.Init(); err != nil {
		log.Fatal("Could not initialize location data:", err)
	}

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
		&model.PropertyView{},
		&model.PropertyStats{},
		&model.Lead{},
		&model.Subscriber{},
	)
	if err != nil {
		log.Printf("Migration warning: %v", err)
	}

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
