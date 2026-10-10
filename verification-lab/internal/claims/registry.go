package claims

// Registry maps lab claim IDs to underlying go test names (documentation + export).
var Registry = map[string][]string{
	"INV-1": {"internal/booking:TestConcurrentUsersUniqueTickets", "integration:TestTwoWorkersNoDoubleSell"},
	"INV-2": {"internal/booking:TestAllocateOneUserOneTicket", "internal/booking:TestConcurrentIdempotentSameUser"},
	"INV-3": {"main:TestBookHandlerIdempotentHTTP", "integration:TestIntegrationBookingIdempotency"},
	"INV-4": {"integration:TestRestartDoesNotMintInventory", "internal/inventory:TestSeedDecision"},
	"INV-5": {"integration:TestReconcileDrift", "internal/reconcile:TestReconcileInventoryDrift"},
	"INV-6": {"main:TestBookEventuallyPersistsViaConsumer", "integration:TestPersistRecovery"},
	"INV-7": {"internal/reconcile:TestReconcileOrphanAllocation", "internal/reconcile:TestReconcileLegitimatePendingNotFalsePositive"},
}
