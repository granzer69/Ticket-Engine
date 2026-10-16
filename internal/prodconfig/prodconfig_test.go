package prodconfig

import "testing"

func TestValidate_productionRequiresAPIKeyAndCORS(t *testing.T) {
	t.Setenv("TICKET_ENV", "production")
	t.Setenv("TICKET_API_KEY", "")
	t.Setenv("TICKET_CORS_ORIGIN", "*")
	if err := Validate(); err == nil {
		t.Fatal("expected error")
	}
	t.Setenv("TICKET_API_KEY", "secret-key")
	t.Setenv("TICKET_CORS_ORIGIN", "https://app.example.com")
	if err := Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestProductionConfig(t *testing.T) {
	TestValidate_productionRequiresAPIKeyAndCORS(t)
}

func TestValidate_devOpen(t *testing.T) {
	t.Setenv("TICKET_ENV", "")
	if err := Validate(); err != nil {
		t.Fatal(err)
	}
}
