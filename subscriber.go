package main

import (
	"encoding/json"
	"log"

	"github.com/ThreeDotsLabs/watermill/message"
)

type CheckTicket struct {
	UserID int
}

func updateTicket(msg *message.Message) error {
	var ticket Ticket
	if err := json.Unmarshal(msg.Payload, &ticket); err != nil {
		return err
	}
	var checkTicket CheckTicket
	if err := db.Model(&Ticket{}).Select("user_id").Where("id = ?", ticket.ID).First(&checkTicket).Error; err != nil {
		return err
	}
	if checkTicket.UserID != dummyUser {
		// Log warning but do NOT crash the server — this is recoverable
		log.Printf("[WARN] Race condition: ticket %d already sold to user %d, skipping update for user %d",
			ticket.ID, checkTicket.UserID, ticket.UserID)
		return nil
	}
	// Mark the ticket taken
	if err := db.Model(&Ticket{}).Where("id = ?", ticket.ID).
		Updates(Ticket{
			UserID: ticket.UserID,
			SoldAt: ticket.SoldAt,
		}).Error; err != nil {
		return err
	}
	return nil
}
