package inventory

import "testing"

func TestSeedDecision(t *testing.T) {
	tests := []struct {
		name      string
		total     int64
		target    int
		wantAct   string
		wantError bool
	}{
		{"empty db", 0, 15000, "insert", false},
		{"exact seed", 15000, 15000, "noop", false},
		{"partial forbidden", 100, 15000, "", true},
		{"over cap", 16000, 15000, "", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			act, err := SeedDecision(tc.total, tc.target)
			if tc.wantError {
				if err == nil {
					t.Fatalf("expected error, got action %q", act)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if act != tc.wantAct {
				t.Fatalf("action %q want %q", act, tc.wantAct)
			}
		})
	}
}
