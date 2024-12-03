// pkg/cron/property_stats.go
package cron

import (
	"estepage_backend/pkg/database"
	"estepage_backend/pkg/email"
	"log"
	"time"

	"github.com/robfig/cron/v3"
)

type PropertyStats struct {
	UserID           uint
	UserEmail        string
	CompanyName      string
	TotalProperties  int64
	TotalViews       int64
	UniqueViews      int64
	TopProperty      string
	TopPropertyViews int64
	LeadCount        int64
}

func InitPropertyStatsCron(emailService *email.EmailService) {
	c := cron.New()

	// Her hafta Pazar günü saat 20:00'de
	_, err := c.AddFunc("0 20 * * 0", func() {
		sendWeeklyPropertyStats(emailService)
	})

	// Her ayın 1'i saat 20:00'de
	_, err = c.AddFunc("0 20 1 * *", func() {
		sendMonthlyPropertyStats(emailService)
	})

	if err != nil {
		log.Printf("Could not initialize property stats cron: %v", err)
		return
	}

	c.Start()
}

func sendWeeklyPropertyStats(emailService *email.EmailService) {
	startDate := time.Now().AddDate(0, 0, -7)
	sendPropertyStats(emailService, startDate, "weekly")
}

func sendMonthlyPropertyStats(emailService *email.EmailService) {
	startDate := time.Now().AddDate(0, -1, 0)
	sendPropertyStats(emailService, startDate, "monthly")
}

func sendPropertyStats(emailService *email.EmailService, startDate time.Time, period string) {
	var stats []PropertyStats

	err := database.GetDB().Raw(`
        SELECT 
            u.id as user_id,
            u.email as user_email,
            u.company_name,
            COUNT(DISTINCT p.id) as total_properties,
            COUNT(pv.id) as total_views,
            COUNT(DISTINCT pv.ip) as unique_views,
            (
                SELECT p2.title 
                FROM properties p2 
                LEFT JOIN property_views pv2 ON p2.id = pv2.property_id
                WHERE p2.user_id = u.id AND pv2.created_at >= ?
                GROUP BY p2.id
                ORDER BY COUNT(pv2.id) DESC
                LIMIT 1
            ) as top_property,
            (
                SELECT COUNT(pv3.id)
                FROM properties p3 
                LEFT JOIN property_views pv3 ON p3.id = pv3.property_id
                WHERE p3.user_id = u.id AND pv3.created_at >= ?
                GROUP BY p3.id
                ORDER BY COUNT(pv3.id) DESC
                LIMIT 1
            ) as top_property_views,
            COUNT(l.id) as lead_count
        FROM users u
        LEFT JOIN properties p ON u.id = p.user_id
        LEFT JOIN property_views pv ON p.id = pv.property_id AND pv.created_at >= ?
        LEFT JOIN leads l ON p.id = l.property_id AND l.created_at >= ?
        GROUP BY u.id
        HAVING total_views > 0
    `, startDate, startDate, startDate, startDate).Scan(&stats).Error

	if err != nil {
		log.Printf("Error fetching property stats: %v", err)
		return
	}

	for _, stat := range stats {
		err := emailService.SendPropertyStats(
			stat.UserEmail,
			stat.CompanyName,
			period,
			stat.TotalProperties,
			stat.TotalViews,
			stat.UniqueViews,
			stat.TopProperty,
			stat.TopPropertyViews,
			stat.LeadCount,
			startDate,
		)
		if err != nil {
			log.Printf("Error sending property stats to %s: %v", stat.UserEmail, err)
		}
	}
}
