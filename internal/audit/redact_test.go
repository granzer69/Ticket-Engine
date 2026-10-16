package audit

import "testing"

func TestLogRedaction_userID(t *testing.T) {
	if UserID(123456789) != "user:***" {
		t.Fatal("expected redacted user id")
	}
}
