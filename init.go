package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync/atomic"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"github.com/go-redis/redis/v8"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

const (
	// queueTicket is the Ticket queue key in redis
	queueTicket = "tickets"
	// hashUser is the user hash key in redis
	hashUser          = "hash:user"
	updateTicketTopic = "ticket.update"
	httpUserHeader    = "X-User-Id"
	dummyUser         = 0
	ticketCount       = 15000
)

var (
	logger = watermill.NewStdLogger(false, false)

	db            *gorm.DB
	redisClient   *redis.Client
	mysqlHost     = getEnv("MYSQL_HOST", "localhost:3306")
	mysqlUser     = getEnv("MYSQL_USER", "root")
	mysqlPassword = getEnv("MYSQL_PASSWORD", "root")
	mysqlDatabase = getEnv("MYSQL_DATABASE", "ticketdb")
	redisHost     = getEnv("REDIS_HOST", "localhost:6379")
	httpPort      = getEnv("HTTP_PORT", "8080")
	pubsub        *gochannel.GoChannel
	pubsubRouter  *message.Router

	// Metrics counters (atomic for concurrency safety)
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
	var err error
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", mysqlUser, mysqlPassword, mysqlHost, mysqlDatabase)
	db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		SkipDefaultTransaction: true, // Skip tx wrapper for single writes — big perf win
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

	if err := db.AutoMigrate(&Ticket{}); err != nil {
		log.Fatalf("failed to run MySQL migrations: %v", err)
	}
}

func initRedis() {
	redisClient = redis.NewClient(&redis.Options{
		Addr:     redisHost,
		DB:       0,
		PoolSize: 500, // Handle 10K concurrent with connection reuse
	})
	if _, err := redisClient.Ping(context.Background()).Result(); err != nil {
		log.Fatalf("failed to connect to Redis at %s: %v", redisHost, err)
	}
	log.Println("Redis connected (pool: 500)")
}

func initPubSub() {
	pubsub = gochannel.NewGoChannel(gochannel.Config{Persistent: true}, logger)
	var err error
	pubsubRouter, err = message.NewRouter(message.RouterConfig{}, logger)
	if err != nil {
		log.Fatal(err)
	}
	pubsubRouter.AddNoPublisherHandler(
		updateTicketTopic+"_handler",
		updateTicketTopic,
		pubsub,
		updateTicket,
	)
}

func prepareData() {
	var count int64
	if err := db.Model(&Ticket{}).Where("user_id = ?", dummyUser).Count(&count).Error; err != nil {
		log.Fatal(err)
	}
	if count < int64(ticketCount) {
		neededTickets := int(int64(ticketCount) - count)
		log.Printf("Inserting %d tickets into MySQL...", neededTickets)
		var newTickets []Ticket
		for i := 0; i < neededTickets; i++ {
			newTickets = append(newTickets, Ticket{UserID: dummyUser})
		}
		if err := db.CreateInBatches(newTickets, 1000).Error; err != nil {
			log.Fatalf("failed to insert dummy tickets: %v", err)
		}
	}

	tickets := make([]Ticket, 0, ticketCount)
	if err := db.Where("user_id = ?", dummyUser).Limit(ticketCount).Find(&tickets).Error; err != nil {
		log.Fatal(err)
	}

	redisTiketLen, err := redisClient.LLen(context.Background(), queueTicket).Result()
	if err != nil {
		log.Fatal(err)
	}
	if redisTiketLen != int64(ticketCount) {
		if err := redisClient.Del(context.Background(), queueTicket).Err(); err != nil {
			log.Fatal(err)
		}

		// Also clear user booking hash so tests are repeatable
		if err := redisClient.Del(context.Background(), hashUser).Err(); err != nil {
			log.Fatal(err)
		}

		// Batch push to Redis
		var ticketIds []interface{}
		for _, t := range tickets {
			ticketIds = append(ticketIds, t.ID)
		}

		for i := 0; i < len(ticketIds); i += 1000 {
			end := i + 1000
			if end > len(ticketIds) {
				end = len(ticketIds)
			}
			if err := redisClient.LPush(context.Background(), queueTicket, ticketIds[i:end]...).Err(); err != nil {
				log.Fatalf("failed to push to Redis: %v", err)
			}
		}

		log.Printf("loaded %d tickets to Redis\n", ticketCount)
	} else {
		log.Printf("Redis already has %d tickets, skipping reload", redisTiketLen)
	}
}

func incrRequests()       { atomic.AddInt64(&totalRequests, 1) }
func incrBookingSuccess() { atomic.AddInt64(&bookingSuccess, 1) }
func incrBookingFail()    { atomic.AddInt64(&bookingFailures, 1) }
