package model

import "gorm.io/gorm"

type Subscription struct {
	gorm.Model
	Name            string  `json:"name" gorm:"not null"`
	Description     string  `json:"description"`
	Price           float64 `json:"price" gorm:"not null"`
	Duration        int     `json:"duration" gorm:"not null"`
	MaxListings     int     `json:"max_listings" gorm:"not null"`
	StripeProductID string  `json:"stripe_product_id"`
	StripePriceID   string  `json:"stripe_price_id"`

	// İlişkiler
	UserSubscriptions []UserSubscription
}

type UserSubscription struct {
	gorm.Model
	UserID         uint   `json:"user_id"`
	SubscriptionID uint   `json:"subscription_id"`
	Status         string `json:"status" gorm:"default:'active'"`
	StripeSubID    string `json:"stripe_subscription_id"`
	ExpiresAt      string `json:"expires_at"`

	// İlişkiler
	User         User         `gorm:"foreignKey:UserID"`
	Subscription Subscription `gorm:"foreignKey:SubscriptionID"`
}
