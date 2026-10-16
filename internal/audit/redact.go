package audit

// UserID returns a log-safe representation of a booking user id (no full id in logs).
func UserID(uid int) string {
	if uid <= 0 {
		return "user:invalid"
	}
	return "user:***"
}
