package seed

import (
	"estepage_backend/internal/model"
	"log"

	"gorm.io/gorm"
)

func SeedSubscriptionPlans(db *gorm.DB) {
	plans := []model.Subscription{
		{
			Name:            "Basic Plan",
			Description:     "For small real estate agencies",
			Price:           29.99,
			Duration:        30,
			MaxListings:     10,
			StripeProductID: "prod_test_basic",
			StripePriceID:   "price_test_basic",
		},
		{
			Name:            "Professional Plan",
			Description:     "For medium-sized agencies",
			Price:           99.99,
			Duration:        30,
			MaxListings:     50,
			StripeProductID: "prod_test_pro",
			StripePriceID:   "price_test_pro",
		},
		{
			Name:            "Enterprise Plan",
			Description:     "For large agencies",
			Price:           299.99,
			Duration:        30,
			MaxListings:     1000,
			StripeProductID: "prod_test_enterprise",
			StripePriceID:   "price_test_enterprise",
		},
	}

	for _, plan := range plans {
		result := db.FirstOrCreate(&plan, model.Subscription{Name: plan.Name})
		if result.Error != nil {
			log.Printf("Error creating plan %s: %v", plan.Name, result.Error)
		}
	}

	log.Println("Subscription plans seeded successfully!")
}
