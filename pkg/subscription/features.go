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
