package claims

import "os"

// IntegrationEnvFromOS returns MYSQL_* and REDIS_HOST for child go test processes.
func IntegrationEnvFromOS() []string {
	return []string{
		"MYSQL_HOST=" + envOr("MYSQL_HOST", "127.0.0.1:3306"),
		"MYSQL_USER=" + envOr("MYSQL_USER", "ticket"),
		"MYSQL_PASSWORD=" + envOr("MYSQL_PASSWORD", "ticket"),
		"MYSQL_DATABASE=" + envOr("MYSQL_DATABASE", "ticketdb"),
		"REDIS_HOST=" + envOr("REDIS_HOST", "127.0.0.1:6379"),
		"REDIS_PASSWORD=" + os.Getenv("REDIS_PASSWORD"),
	}
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
