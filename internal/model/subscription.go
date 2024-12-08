package model

import (
	"time"

	"gorm.io/gorm"
)

type Subscription struct {
	gorm.Model
	Name            string  `json:"name" gorm:"not null"`
	Description     string  `json:"description"`
	Price           float64 `json:"price" gorm:"not null"`
	Duration        int     `json:"duration" gorm:"not null"`
	MaxListings     int     `json:"max_listings" gorm:"not null"`
	ThemeLimit      int     `json:"theme_limit" gorm:"not null;default:1"`
	StripeProductID string  `json:"stripe_product_id"`
	StripePriceID   string  `json:"stripe_price_id"`
}

type UserSubscription struct {
	gorm.Model
	UserID           uint      `json:"user_id" gorm:"not null"`
	StripePlanID     string    `json:"stripe_plan_id"`
	Status           string    `json:"status" gorm:"default:'active'"`
	StripeSubID      string    `json:"stripe_subscription_id"`
	ExpiresAt        string    `json:"expires_at"`
	CancellationDate time.Time `json:"cancellation_date"`
	User             User      `gorm:"foreignKey:UserID"`
}
