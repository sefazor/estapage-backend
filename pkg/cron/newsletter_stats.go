package cron

import (
	"estepage_backend/pkg/database"
	"estepage_backend/pkg/email"
	"log"
	"time"

	"github.com/robfig/cron/v3"
)

func InitNewsletterCron() {
	c := cron.New()

	_, err := c.AddFunc("0 19 * * *", func() {
		sendDailyNewsletterStats()
	})

	if err != nil {
		log.Printf("Could not initialize newsletter cron: %v", err)
		return
	}

	c.Start()
}

func sendDailyNewsletterStats() {
	today := time.Now().Format("2006-01-02")

	var stats []struct {
		UserID          uint
		UserEmail       string
		CompanyName     string
		SubscriberCount int64
	}

	err := database.DB.Raw(`
        SELECT 
            u.id as user_id,
            u.email as user_email,
            u.company_name,
            COUNT(s.id) as subscriber_count
        FROM users u
        LEFT JOIN subscribers s ON u.id = s.user_id
        WHERE DATE(s.subscribed_at) = ?
        GROUP BY u.id
        HAVING COUNT(s.id) > 0
    `, today).Scan(&stats).Error

	if err != nil {
		log.Printf("Error fetching newsletter stats: %v", err)
		return
	}

	for _, stat := range stats {
		if email.GlobalEmailService != nil {
			err := email.GlobalEmailService.SendDailyNewsletterStats(
				stat.UserEmail,
				stat.CompanyName,
				stat.SubscriberCount,
				time.Now(),
			)
			if err != nil {
				log.Printf("Error sending newsletter stats to %s: %v", stat.UserEmail, err)
			}
		}
	}
}
