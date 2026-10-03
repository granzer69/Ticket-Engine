package inventory

import "fmt"

// SeedDecision returns the action to take when seeding inventory.
// It refuses to top up partial row counts (restart-safe).
func SeedDecision(totalRows int64, target int) (action string, err error) {
	if target <= 0 {
		return "", fmt.Errorf("invalid target inventory: %d", target)
	}
	switch {
	case totalRows == 0:
		return "insert", nil
	case totalRows == int64(target):
		return "noop", nil
	case totalRows > int64(target):
		return "", fmt.Errorf("database has %d tickets, exceeds seed cap %d", totalRows, target)
	default:
		return "", fmt.Errorf("database has %d tickets, expected 0 or %d; refusing to top up inventory", totalRows, target)
	}
}
