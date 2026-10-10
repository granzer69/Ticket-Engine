package security

import "crypto/subtle"

// APIKeyValid reports whether provided satisfies required API key policy.
// If required is empty, authentication is disabled (development default).
func APIKeyValid(provided, required string) bool {
	if required == "" {
		return true
	}
	if provided == "" {
		return false
	}
	if len(provided) != len(required) {
		// Avoid short-circuit timing leak; compare against self.
		subtle.ConstantTimeCompare([]byte(required), []byte(required))
		return false
	}
	return subtle.ConstantTimeCompare([]byte(provided), []byte(required)) == 1
}
