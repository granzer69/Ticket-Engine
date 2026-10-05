package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync/atomic"

	"ticketengine/internal/booking"
	"ticketengine/internal/persist"
	"ticketengine/internal/security"
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
	origin := getEnv("TICKET_CORS_ORIGIN", "*")
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, X-User-Id, X-API-Key")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next(w, r)
	}
}

func apiKeyMiddleware(next http.HandlerFunc) http.HandlerFunc {
	required := os.Getenv("TICKET_API_KEY")
	if required == "" {
		return next
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if !security.APIKeyValid(r.Header.Get("X-API-Key"), required) {
			writeJSON(w, http.StatusUnauthorized, BookingResponse{
				Status:  "error",
				Message: "Unauthorized",
			})
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
	if err != nil || uid <= 0 {
		incrBookingFail()
		writeJSON(w, http.StatusBadRequest, BookingResponse{
			Status:  "error",
			Message: "Invalid user ID",
		})
		return
	}

	ctx := context.Background()
	allocResult, err := ticketAllocator.Allocate(ctx, uid)
	if err != nil {
		incrBookingFail()
		log.Printf("booking allocate failed: %v", err)
		writeJSON(w, http.StatusServiceUnavailable, BookingResponse{
			Status:  "error",
			Message: "Booking service temporarily unavailable",
		})
		return
	}
	if allocResult.SoldOut {
		incrBookingFail()
		writeJSON(w, http.StatusNotFound, BookingResponse{
			Status:  "error",
			Message: "No tickets left",
		})
		return
	}
	ticketID := allocResult.TicketID

	if allocResult.Replay {
		if err := ensurePersisted(ctx, ticketID, uid); err != nil {
			incrBookingFail()
			log.Printf("replay persist enqueue failed: %v", err)
			writeJSON(w, http.StatusServiceUnavailable, BookingResponse{
				Status:  "error",
				Message: "Booking service temporarily unavailable",
			})
			return
		}
		writeBookingSuccess(w, ticketID, uid)
		return
	}

	writeBookingSuccess(w, ticketID, uid)
}

func ensurePersisted(ctx context.Context, ticketID, userID int) error {
	redisTicket, ok, err := booking.TicketForUser(ctx, redisClient, userID)
	if err != nil {
		return err
	}
	if ok && redisTicket == ticketID {
		return nil
	}
	return persist.EnqueueReplay(ctx, redisClient, ticketID, userID)
}

func writeBookingSuccess(w http.ResponseWriter, ticketID, uid int) {
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
		TotalRequests:   atomic.LoadInt64(&totalRequests),
		BookingSuccess:  atomic.LoadInt64(&bookingSuccess),
		BookingFailures: atomic.LoadInt64(&bookingFailures),
	})
}
