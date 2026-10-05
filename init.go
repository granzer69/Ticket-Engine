package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync/atomic"

	"github.com/go-redis/redis/v8"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

const (
	queueTicket    = "tickets"
	hashUser       = "hash:user"
	httpUserHeader = "X-User-Id"
	ticketCount    = 15000
)

var (
	db            *gorm.DB
	redisClient   *redis.Client
	mysqlHost     = getEnv("MYSQL_HOST", "localhost:3306")
	mysqlUser     = getEnv("MYSQL_USER", "root")
	mysqlPassword = getEnv("MYSQL_PASSWORD", "root")
	mysqlDatabase = getEnv("MYSQL_DATABASE", "ticketdb")
	redisHost     = getEnv("REDIS_HOST", "localhost:6379")
	httpPort      = getEnv("HTTP_PORT", "8080")

	totalRequests   int64
	bookingSuccess  int64
	bookingFailures int64
)

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}
	return fallback
}

func initMySQL() {
	if db != nil {
		return
	}
	var err error
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", mysqlUser, mysqlPassword, mysqlHost, mysqlDatabase)
	db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		SkipDefaultTransaction: true,
	})
	if err != nil {
		log.Fatalf("failed to connect to MySQL: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("failed to get MySQL db object: %v", err)
	}
	sqlDB.SetMaxOpenConns(200)
	sqlDB.SetMaxIdleConns(100)
	log.Println("DB connected (pool: 200 open, 100 idle)")

	applyMigrations()
	if err := db.AutoMigrate(&Ticket{}); err != nil {
		log.Fatalf("failed to run GORM schema sync: %v", err)
	}
}

func initRedis() {
	if redisClient != nil {
		return
	}
	redisClient = redis.NewClient(&redis.Options{
		Addr:     redisHost,
		DB:       0,
		PoolSize: 500,
	})
	if _, err := redisClient.Ping(context.Background()).Result(); err != nil {
		log.Fatalf("failed to connect to Redis at %s: %v", redisHost, err)
	}
	log.Println("Redis connected (pool: 500)")
}

func prepareData() {
	prepareRuntimeData()
}

func incrRequests()       { atomic.AddInt64(&totalRequests, 1) }
func incrBookingSuccess() { atomic.AddInt64(&bookingSuccess, 1) }
func incrBookingFail()    { atomic.AddInt64(&bookingFailures, 1) }
