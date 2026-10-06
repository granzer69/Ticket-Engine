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

benchmark-smoke:
	@command -v k6 >/dev/null || (echo "k6 not installed; see https://k6.io/docs/get-started/installation/" && exit 1)
	k6 run --vus 20 --duration 15s test.js
