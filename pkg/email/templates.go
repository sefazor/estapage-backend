package email

import (
	"embed"
	"html/template"
)

//go:embed templates/*.html
var templateFS embed.FS

// loadTemplates email template'lerini yükler
func loadTemplates() (*template.Template, error) {
	return template.ParseFS(templateFS, "templates/*.html")
}
