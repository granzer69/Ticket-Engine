.PHONY: test test-race test-integration compose-config seed

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
