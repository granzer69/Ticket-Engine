package main

import (
	"log"
	"os"

	"ticketengine/verification-lab/internal/api"
	"ticketengine/verification-lab/internal/metrics"
	"ticketengine/verification-lab/internal/runner"
	"ticketengine/verification-lab/internal/target"
)

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func main() {
	rawTarget := env("VERILAB_TARGET", "http://127.0.0.1:8080")
	base, err := target.BaseURL(rawTarget)
	if err != nil {
		log.Fatalf("target: %v", err)
	}
	mysqlHost := env("MYSQL_HOST", "127.0.0.1:3306")
	dsn := metrics.DSN(mysqlHost, env("MYSQL_USER", "ticket"), env("MYSQL_PASSWORD", "ticket"), env("MYSQL_DATABASE", "ticketdb"))

	eng := runner.New(runner.Config{
		TargetBase: base,
		APIKey:     os.Getenv("TICKET_API_KEY"),
		RedisAddr:  env("REDIS_HOST", "127.0.0.1:6379"),
		MySQLDSN:   dsn,
	})
	hub := metrics.NewHub(eng.Collector(), eng)
	eng.Hub = hub

	srv := &api.Server{
		Addr:   env("VERILAB_HTTP", ":8090"),
		Engine: eng,
		Hub:    hub,
	}
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
