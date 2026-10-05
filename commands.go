package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"ticketengine/internal/reconcile"
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
	rep, err := RunReconciliation(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "RECONCILE ERROR: %v\n", err)
		os.Exit(reconcile.ExitOperational)
	}
	fmt.Print(rep.Format())
	if rep.HasDrift() {
		os.Exit(reconcile.ExitDrift)
	}
}
