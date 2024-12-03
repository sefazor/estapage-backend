// pkg/email/email.go
package email

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

type EmailService struct {
	apiKey    string
	from      string
	templates *template.Template
}

type EmailData struct {
	From    string `json:"from"`
	To      string `json:"to"`
	Subject string `json:"subject"`
	Html    string `json:"html"`
}

// Template data structures
type WelcomeEmailData struct {
	Name string
}

type LeadNotificationData struct {
	PropertyTitle string
	LeadName      string
	LeadEmail     string
	LeadPhone     string
	LeadMessage   string
}

type SubscriptionEmailData struct {
	CompanyName string
	PlanName    string
	Duration    int
	Price       float64
	Currency    string
	MaxListings int
	ExpiresAt   time.Time
	IsRenewal   bool
}

type SubscriptionCancelledData struct {
	CompanyName string
	PlanName    string
	ExpiresAt   time.Time
}

type SubscriptionExpiryWarningData struct {
	CompanyName string
	PlanName    string
	DaysLeft    int
	ExpiryDate  time.Time
}

type PasswordResetData struct {
	ResetLink string
}

type PasswordChangedData struct {
	Email string
}

type DailyNewsletterStatsData struct {
	CompanyName     string
	SubscriberCount int64
	Date            time.Time
}

type PropertyStatsData struct {
	CompanyName      string
	Period           string
	TotalProperties  int64
	TotalViews       int64
	UniqueViews      int64
	TopProperty      string
	TopPropertyViews int64
	LeadCount        int64
	StartDate        time.Time
}

func NewEmailService(apiKey string) (*EmailService, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("resend API key is required")
	}

	templates, err := loadTemplates()
	if err != nil {
		return nil, fmt.Errorf("error loading email templates: %v", err)
	}

	return &EmailService{
		apiKey:    apiKey,
		from:      "EstaPage <noreply@estapage.com>",
		templates: templates,
	}, nil
}

func (s *EmailService) sendTemplateEmail(to, subject, templateName string, data interface{}) error {
	var body bytes.Buffer
	if err := s.templates.ExecuteTemplate(&body, templateName, data); err != nil {
		return fmt.Errorf("template execution error: %v", err)
	}

	emailData := EmailData{
		From:    s.from,
		To:      to,
		Subject: subject,
		Html:    body.String(),
	}

	jsonData, err := json.Marshal(emailData)
	if err != nil {
		return fmt.Errorf("error marshaling email data: %v", err)
	}

	log.Printf("Sending email to: %s with data: %s", to, string(jsonData))

	req, err := http.NewRequest("POST", "https://api.resend.com/emails", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending email: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response body: %v", err)
	}

	log.Printf("Resend API response: Status: %d, Body: %s", resp.StatusCode, string(respBody))

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("resend API error: %s", string(respBody))
	}

	return nil
}

// Email sending methods
func (s *EmailService) SendWelcomeEmail(email, name string) error {
	data := WelcomeEmailData{
		Name: name,
	}
	return s.sendTemplateEmail(email, "Welcome to EstePage! 🎉", "welcome.html", data)
}

func (s *EmailService) SendLeadNotificationEmail(
	agentEmail, propertyTitle, leadName, leadEmail, leadPhone, leadMessage string,
) error {
	data := LeadNotificationData{
		PropertyTitle: propertyTitle,
		LeadName:      leadName,
		LeadEmail:     leadEmail,
		LeadPhone:     leadPhone,
		LeadMessage:   leadMessage,
	}
	return s.sendTemplateEmail(agentEmail, "New Lead for Your Property! 📋", "lead_notification.html", data)
}

func (s *EmailService) SendSubscriptionStartedEmail(
	email string,
	companyName string,
	planName string,
	duration int,
	price float64,
	currency string,
	maxListings int,
	expiresAt time.Time,
	isRenewal bool,
) error {
	data := SubscriptionEmailData{
		CompanyName: companyName,
		PlanName:    planName,
		Duration:    duration,
		Price:       price,
		Currency:    currency,
		MaxListings: maxListings,
		ExpiresAt:   expiresAt,
		IsRenewal:   isRenewal,
	}

	subject := "Welcome to EstePage Premium! 🎉"
	if isRenewal {
		subject = "Your EstePage Subscription Has Been Renewed 🔄"
	}

	return s.sendTemplateEmail(email, subject, "subscription_started.html", data)
}

func (s *EmailService) SendSubscriptionCancelledEmail(email, companyName, planName string, expiresAt time.Time) error {
	data := SubscriptionCancelledData{
		CompanyName: companyName,
		PlanName:    planName,
		ExpiresAt:   expiresAt,
	}
	return s.sendTemplateEmail(email, "Your Subscription Has Been Cancelled", "subscription_cancelled.html", data)
}

func (s *EmailService) SendSubscriptionExpiryWarning(
	email, companyName, planName string,
	expiryDate time.Time,
	daysLeft int,
) error {
	data := SubscriptionExpiryWarningData{
		CompanyName: companyName,
		PlanName:    planName,
		DaysLeft:    daysLeft,
		ExpiryDate:  expiryDate,
	}
	return s.sendTemplateEmail(
		email,
		fmt.Sprintf("Your Subscription Expires in %d Days ⚠️", daysLeft),
		"subscription_expiry_warning.html",
		data,
	)
}

func (s *EmailService) SendPasswordResetEmail(email, resetToken string) error {
	data := PasswordResetData{
		ResetLink: fmt.Sprintf("https://estepage.com/reset-password?token=%s", resetToken),
	}
	return s.sendTemplateEmail(email, "Reset Your Password 🔒", "password_reset.html", data)
}

func (s *EmailService) SendPasswordChangedEmail(email string) error {
	data := PasswordChangedData{
		Email: email,
	}
	return s.sendTemplateEmail(email, "Your Password Has Been Changed 🔐", "password_changed.html", data)
}

func (s *EmailService) SendDailyNewsletterStats(email, companyName string, subscriberCount int64, date time.Time) error {
	data := DailyNewsletterStatsData{
		CompanyName:     companyName,
		SubscriberCount: subscriberCount,
		Date:            date,
	}
	return s.sendTemplateEmail(email, "Your Daily Newsletter Statistics 📊", "daily_newsletter_stats.html", data)
}

func (s *EmailService) SendPropertyStats(
	email, companyName, period string,
	totalProperties, totalViews, uniqueViews int64,
	topProperty string, topPropertyViews, leadCount int64,
	startDate time.Time,
) error {
	data := PropertyStatsData{
		CompanyName:      companyName,
		Period:           period,
		TotalProperties:  totalProperties,
		TotalViews:       totalViews,
		UniqueViews:      uniqueViews,
		TopProperty:      topProperty,
		TopPropertyViews: topPropertyViews,
		LeadCount:        leadCount,
		StartDate:        startDate,
	}
	subject := fmt.Sprintf("Your %s Property Statistics 📊", strings.Title(period))
	return s.sendTemplateEmail(email, subject, "property_stats.html", data)
}
