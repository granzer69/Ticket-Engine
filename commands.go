package main

import (
	"context"
	"log"
)

func runSeedCommand() {
	log.Println("Running inventory seed (explicit command)...")
	initMySQL()
	initRedis()
	if err := SeedInventory(); err != nil {
		log.Fatalf("seed failed: %v", err)
	}
	log.Println("Seed complete.")
}

func runReconcileCommand() {
	initMySQL()
	initRedis()
	if err := RunReconciliation(context.Background()); err != nil {
		log.Fatalf("reconcile failed: %v", err)
	}
	log.Println("reconcile complete")
}
