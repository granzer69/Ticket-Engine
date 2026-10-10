package main

import (
	"os"
	"strconv"
)

func mysqlPoolLimits() (maxOpen, maxIdle int) {
	role := getEnv("TICKET_ENGINE_ROLE", "serve")
	if role == "worker" {
		maxOpen = envInt("TICKET_MYSQL_MAX_OPEN", 50)
		maxIdle = envInt("TICKET_MYSQL_MAX_IDLE", 25)
		return maxOpen, maxIdle
	}
	return envInt("TICKET_MYSQL_MAX_OPEN", 200), envInt("TICKET_MYSQL_MAX_IDLE", 100)
}

func envInt(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 1 {
		return def
	}
	return n
}
