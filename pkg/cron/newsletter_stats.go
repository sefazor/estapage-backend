// pkg/cron/newsletter_stats.go

package cron

import (
	"estepage_backend/pkg/database"
	"estepage_backend/pkg/email"
	"log"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

var (
	lastRunTime time.Time
	mutex       sync.Mutex
)

func InitNewsletterCron() {
	c := cron.New()

	// Her gün saat 19:00'da çalışacak
	_, err := c.AddFunc("0 19 * * *", func() {
		mutex.Lock()
		defer mutex.Unlock()

		// Son çalışma zamanını kontrol et
		if time.Since(lastRunTime) < 23*time.Hour {
			log.Printf("Newsletter stats already sent today, skipping...")
			return
		}

		sendDailyNewsletterStats()
		lastRunTime = time.Now()
	})

	if err != nil {
		log.Printf("Could not initialize newsletter cron: %v", err)
		return
	}

	c.Start()
	log.Printf("Newsletter cron initialized successfully")
}

func sendDailyNewsletterStats() {
	today := time.Now().Format("2006-01-02")
	log.Printf("Running newsletter stats for date: %s", today)

	var stats []struct {
		UserID          uint
		UserEmail       string
		CompanyName     string
		SubscriberCount int64
	}

	// O günün tarihine ait kayıtları çek
	err := database.DB.Raw(`
        SELECT 
            u.id as user_id,
            u.email as user_email,
            u.company_name,
            COUNT(s.id) as subscriber_count
        FROM users u
        LEFT JOIN newsletter_subscribers s ON u.id = s.user_id
        WHERE DATE(s.subscribed_at) = ?
        GROUP BY u.id
        HAVING COUNT(s.id) > 0
    `, today).Scan(&stats).Error

	if err != nil {
		log.Printf("Error fetching newsletter stats: %v", err)
		return
	}

	log.Printf("Found %d users with new subscribers", len(stats))

	for _, stat := range stats {
		if email.GlobalEmailService != nil {
			log.Printf("Sending stats email to %s (Company: %s, Subscribers: %d)",
				stat.UserEmail, stat.CompanyName, stat.SubscriberCount)

			err := email.GlobalEmailService.SendDailyNewsletterStats(
				stat.UserEmail,
				stat.CompanyName,
				stat.SubscriberCount,
				time.Now(),
			)
			if err != nil {
				log.Printf("Error sending newsletter stats to %s: %v", stat.UserEmail, err)
			} else {
				log.Printf("Successfully sent stats email to %s", stat.UserEmail)
			}
		}
	}
}
