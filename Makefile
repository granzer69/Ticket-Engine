.PHONY: test test-race test-integration test-verilab build-verilab-ui compose-config seed run-api run-worker run-verilab dev verify-smoke golden-evidence

test:
	go test ./...

test-race:
	go test -race ./...

test-integration:
	go test -tags=integration ./integration/...

test-verilab:
	go test ./verification-lab/...

build-verilab-ui:
	cd verification-lab/web && npm ci && npm run build

run-verilab:
	@test -n "$(VERILAB_WEB_DIST)" || export VERILAB_WEB_DIST="$$(pwd)/verification-lab/web/dist"; \
	go run ./verification-lab/cmd/verilab

verify-smoke:
	@curl -sf -X POST http://127.0.0.1:8090/api/runs -H 'Content-Type: application/json' -d '{"preset":"smoke"}' | head -c 200; echo

golden-evidence:
	@./scripts/golden-evidence.sh

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
