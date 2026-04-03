package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// JSON response types for structured API output
type BookingResponse struct {
	Status   string `json:"status"`
	TicketID int    `json:"ticket_id,omitempty"`
	UserID   int    `json:"user_id,omitempty"`
	Message  string `json:"message,omitempty"`
}

type CountResponse struct {
	Remaining int64 `json:"remaining"`
}

type MetricsResponse struct {
	TotalRequests   int64 `json:"total_requests"`
	BookingSuccess  int64 `json:"booking_success"`
	BookingFailures int64 `json:"booking_failures"`
}

func enableCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, X-User-Id")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next(w, r)
	}
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func ticketHandler(w http.ResponseWriter, r *http.Request) {
	incrRequests()

	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, BookingResponse{
			Status:  "error",
			Message: "Method Not Allowed - Use POST",
		})
		return
	}

	uidStr := r.Header.Get(httpUserHeader)
	if uidStr == "" {
		incrBookingFail()
		writeJSON(w, http.StatusUnauthorized, BookingResponse{
			Status:  "error",
			Message: "Missing X-User-Id header",
		})
		return
	}
	uid, err := strconv.Atoi(uidStr)
	if err != nil {
		incrBookingFail()
		writeJSON(w, http.StatusBadRequest, BookingResponse{
			Status:  "error",
			Message: "Invalid user ID",
		})
		return
	}

	// Check whether the current user has already bought a ticket (atomic via Redis HINCRBY)
	if result, err := redisClient.HIncrBy(context.Background(), hashUser, uidStr, 1).Result(); err != nil || result != 1 {
		incrBookingFail()
		writeJSON(w, http.StatusTooManyRequests, BookingResponse{
			Status:  "error",
			Message: "User already booked a ticket",
		})
		return
	}

	// Atomically pop a ticket from the Redis queue — this is the critical path
	ticketIdStr, err := redisClient.LPop(context.Background(), queueTicket).Result()
	if err != nil {
		incrBookingFail()
		writeJSON(w, http.StatusNotFound, BookingResponse{
			Status:  "error",
			Message: "No tickets left",
		})
		return
	}
	ticketID, err := strconv.Atoi(ticketIdStr)
	if err != nil {
		incrBookingFail()
		writeJSON(w, http.StatusInternalServerError, BookingResponse{
			Status:  "error",
			Message: "Internal Server Error",
		})
		return
	}

	// Publish to async worker for MySQL persistence — keeps response fast
	soldAt := time.Now()
	payload, err := json.Marshal(&Ticket{
		ID:     ticketID,
		UserID: uid,
		SoldAt: &soldAt,
	})
	if err != nil {
		incrBookingFail()
		log.Printf("Failed to marshal ticket payload: %v", err)
		writeJSON(w, http.StatusInternalServerError, BookingResponse{
			Status:  "error",
			Message: "Internal Server Error",
		})
		return
	}
	msg := message.NewMessage(watermill.NewUUID(), payload)
	if err := pubsub.Publish(updateTicketTopic, msg); err != nil {
		incrBookingFail()
		log.Printf("Failed to publish ticket update: %v", err)
		writeJSON(w, http.StatusInternalServerError, BookingResponse{
			Status:  "error",
			Message: "Internal Server Error",
		})
		return
	}

	incrBookingSuccess()
	writeJSON(w, http.StatusOK, BookingResponse{
		Status:   "success",
		TicketID: ticketID,
		UserID:   uid,
		Message:  fmt.Sprintf("Ticket %d booked for user %d", ticketID, uid),
	})
}

func ticketCountHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, BookingResponse{
			Status:  "error",
			Message: "Method Not Allowed - Use GET",
		})
		return
	}

	count, err := redisClient.LLen(context.Background(), queueTicket).Result()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, BookingResponse{
			Status:  "error",
			Message: "Failed to query ticket count",
		})
		return
	}

	writeJSON(w, http.StatusOK, CountResponse{Remaining: count})
}

func metricsHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, MetricsResponse{
		TotalRequests:   totalRequests,
		BookingSuccess:  bookingSuccess,
		BookingFailures: bookingFailures,
	})
}
