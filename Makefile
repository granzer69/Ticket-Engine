.PHONY: test test-race test-integration compose-config seed run-api run-worker dev

test:
	go test ./...

test-race:
	go test -race ./...

test-integration:
	go test -tags=integration ./integration/...

compose-config:
	docker compose config

seed:
	go run . seed

run-api:
	go run .

run-worker:
	go run . worker

# API and persist worker are separate processes; serve alone does not drain the bookings stream.
dev:
	trap 'kill 0' INT TERM; go run . worker & go run . & wait
