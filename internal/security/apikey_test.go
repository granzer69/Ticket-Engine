package security

import "testing"

func TestAPIKeyValid_DisabledWhenRequiredEmpty(t *testing.T) {
	if !APIKeyValid("", "") {
		t.Fatal("expected open when required empty")
	}
	if !APIKeyValid("anything", "") {
		t.Fatal("expected open when required empty")
	}
}

func TestAPIKeyValid_MissingProvided(t *testing.T) {
	if APIKeyValid("", "required-secret") {
		t.Fatal("expected reject when provided empty")
	}
}

func TestAPIKeyValid_Invalid(t *testing.T) {
	if APIKeyValid("wrong-key-value", "required-secret") {
		t.Fatal("expected reject for wrong key")
	}
}

func TestAPIKeyValid_Valid(t *testing.T) {
	key := "test-api-key-12345"
	if !APIKeyValid(key, key) {
		t.Fatal("expected accept for matching key")
	}
}
