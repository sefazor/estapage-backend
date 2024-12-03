// pkg/email/service.go
package email

var GlobalEmailService *EmailService

func InitEmailService(apiKey string) error {
	service, err := NewEmailService(apiKey)
	if err != nil {
		return err
	}
	GlobalEmailService = service
	return nil
}
