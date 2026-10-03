package main

import "time"

// Ticket is the ticket type
type Ticket struct {
	ID        int        `gorm:"primaryKey"`
	UserID    *int       `gorm:"uniqueIndex"`
	State     string     `gorm:"size:16;default:available;index"`
	CreatedAt int64      `gorm:"autoCreateTime:milli"`
	SoldAt    *time.Time
}
