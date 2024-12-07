package subscription

type PlanType string
type Feature string

const (
	FreePlan  PlanType = "FREE"
	ProPlan   PlanType = "PRO"
	ElitePlan PlanType = "ELITE"
)

const (
	LeadForm        Feature = "lead_form"
	NewsletterForm  Feature = "newsletter_form"
	WhatsAppButton  Feature = "whatsapp_button"
	MaxListings     Feature = "max_listings"
	MaxImages       Feature = "max_images"
	EmailSupport    Feature = "email_support"
	PrioritySupport Feature = "priority_support"
)

type PlanLimits struct {
	MaxListings      int
	MaxImagesPerList int
	AllowedFeatures  map[Feature]bool
}

var PlanFeatures = map[PlanType]PlanLimits{
	FreePlan: {
		MaxListings:      1,
		MaxImagesPerList: 5,
		AllowedFeatures: map[Feature]bool{
			LeadForm:        false,
			NewsletterForm:  false,
			WhatsAppButton:  false,
			EmailSupport:    false,
			PrioritySupport: false,
		},
	},
	ProPlan: {
		MaxListings:      25,
		MaxImagesPerList: 16,
		AllowedFeatures: map[Feature]bool{
			LeadForm:        true,
			NewsletterForm:  true,
			WhatsAppButton:  true,
			EmailSupport:    true,
			PrioritySupport: false,
		},
	},
	ElitePlan: {
		MaxListings:      100,
		MaxImagesPerList: 16,
		AllowedFeatures: map[Feature]bool{
			LeadForm:        true,
			NewsletterForm:  true,
			WhatsAppButton:  true,
			EmailSupport:    true,
			PrioritySupport: true,
		},
	},
}

// Helper functions
func CanUseFeature(plan PlanType, feature Feature) bool {
	limits, exists := PlanFeatures[plan]
	if !exists {
		return false
	}
	return limits.AllowedFeatures[feature]
}

func GetPlanLimits(plan PlanType) PlanLimits {
	return PlanFeatures[plan]
}

func DeterminePlanType(stripePlanID string) PlanType {
	switch stripePlanID {
	case "price_1QT3IEJuNU9LluRUWytR6JS5":
		return ProPlan
	case "price_1QT3IaJuNU9LluRUg21Cv7QU":
		return ElitePlan
	default:
		return FreePlan
	}
}

// GetPlanMaxListings direkt plan tipinden maksimum listing sayısını döndürür
func GetPlanMaxListings(planType PlanType) int {
	return PlanFeatures[planType].MaxListings
}

// GetPlanMaxImages direkt plan tipinden maksimum resim sayısını döndürür
func GetPlanMaxImages(planType PlanType) int {
	return PlanFeatures[planType].MaxImagesPerList
}

// GetPlanNameFromStripeID stripe plan ID'sinden insan tarafından okunabilir plan adını döndürür
func GetPlanNameFromStripeID(stripePlanID string) string {
	planType := DeterminePlanType(stripePlanID)
	switch planType {
	case ProPlan:
		return "Pro Plan"
	case ElitePlan:
		return "Elite Plan"
	default:
		return "Free Plan"
	}
}

// CalculateRemainingListings kalan listing sayısını hesaplar
func CalculateRemainingListings(planType PlanType, currentListings int) int {
	maxListings := GetPlanMaxListings(planType)
	return maxListings - currentListings
}

// IsPlanFeatureEnabled plan tipine göre bir özelliğin aktif olup olmadığını kontrol eder
func IsPlanFeatureEnabled(planType PlanType, feature Feature) bool {
	if limits, exists := PlanFeatures[planType]; exists {
		if enabled, ok := limits.AllowedFeatures[feature]; ok {
			return enabled
		}
	}
	return false
}
