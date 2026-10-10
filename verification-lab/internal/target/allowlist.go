package target

import (
	"fmt"
	"net/url"
	"strings"
)

// BaseURL validates and normalizes an allowlisted HTTP target (localhost only).
func BaseURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("target: empty URL")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("target: parse: %w", err)
	}
	if u.Scheme != "http" {
		return "", fmt.Errorf("target: only http scheme allowed")
	}
	host := strings.ToLower(u.Hostname())
	switch host {
	case "127.0.0.1", "localhost", "::1":
	default:
		return "", fmt.Errorf("target: host %q not allowlisted", host)
	}
	u.Path = strings.TrimSuffix(u.Path, "/")
	u.RawQuery = ""
	u.Fragment = ""
	return u.String(), nil
}
