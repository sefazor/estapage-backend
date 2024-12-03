package cron

import (
	"estepage_backend/internal/model"
	"estepage_backend/pkg/database"
	"estepage_backend/pkg/email"
	"log"
	"time"

	"github.com/robfig/cron/v3"
)

func InitSubscriptionExpiryCron() {
	c := cron.New()

	_, err := c.AddFunc("0 9 * * *", func() {
		checkExpiringSubscriptions()
	})

	if err != nil {
		log.Printf("Could not initialize subscription expiry cron: %v", err)
		return
	}

	c.Start()
}

func checkExpiringSubscriptions() {
	log.Println("Checking for expiring subscriptions...")

	warningDays := []int{7, 3}

	for _, days := range warningDays {
		var subs []model.UserSubscription
		targetDate := time.Now().AddDate(0, 0, days).Format("2006-01-02")

		err := database.DB.Where("DATE(expires_at) = ? AND status = ?", targetDate, "active").
			Preload("User").
			Preload("Subscription").
			Find(&subs).Error

		if err != nil {
			log.Printf("Error fetching expiring subscriptions: %v", err)
			continue
		}

		log.Printf("Found %d subscriptions expiring in %d days", len(subs), days)

		for _, sub := range subs {
			if email.GlobalEmailService != nil {
				expiresAt, err := time.Parse(time.RFC3339, sub.ExpiresAt)
				if err != nil {
					log.Printf("Error parsing expiry date for subscription %d: %v", sub.ID, err)
					continue
				}

				err = email.GlobalEmailService.SendSubscriptionExpiryWarning(
					sub.User.Email,
					sub.User.CompanyName,
					sub.Subscription.Name,
					expiresAt,
					days,
				)
				if err != nil {
					log.Printf("Error sending expiry warning to %s: %v", sub.User.Email, err)
				} else {
					log.Printf("Sent expiry warning to %s for subscription expiring in %d days", sub.User.Email, days)
				}
			}
		}
	}
}
