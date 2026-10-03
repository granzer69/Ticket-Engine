package main

import (
	"context"
	"log"

	"ticketengine/internal/booking"
)

var ticketAllocator *booking.Allocator

func initBookingAllocator() {
	alloc, err := booking.NewAllocator(context.Background(), redisClient)
	if err != nil {
		log.Fatalf("failed to init booking allocator: %v", err)
	}
	ticketAllocator = alloc
	log.Println("Booking allocator ready (Redis Lua)")
}
