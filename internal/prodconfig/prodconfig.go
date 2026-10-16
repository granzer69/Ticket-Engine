package prodconfig

import (
	"fmt"
	"os"
	"strings"
)

// Validate fails fast when TICKET_ENV=production and required settings are missing or unsafe.
func Validate() error {
	if strings.ToLower(strings.TrimSpace(os.Getenv("TICKET_ENV"))) != "production" {
		return nil
	}
	if strings.TrimSpace(os.Getenv("TICKET_API_KEY")) == "" {
		return fmt.Errorf("production: TICKET_API_KEY is required when TICKET_ENV=production")
	}
	origin := strings.TrimSpace(os.Getenv("TICKET_CORS_ORIGIN"))
	if origin == "" || origin == "*" {
		return fmt.Errorf("production: TICKET_CORS_ORIGIN must be an explicit origin (not empty or *)")
	}
	return nil
}

// IsProduction reports whether strict production profile is active.
func IsProduction() bool {
	return strings.ToLower(strings.TrimSpace(os.Getenv("TICKET_ENV"))) == "production"
}
