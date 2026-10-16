# Verification Lab

Control plane and UI for load tests, phase gates (P1–P7), booking invariants (INV), benchmarks, and safe failure scenarios against a running Ticket Engine stack.

## Prerequisites

- Ticket Engine API on `VERILAB_TARGET` (default `http://127.0.0.1:8080`)
- `go run . worker` running (MySQL persistence)
- Redis + MySQL reachable (same env as engine)
- Built UI: `cd verification-lab/web && npm ci && npm run build`

## Run verilab

```bash
export VERILAB_TARGET=http://127.0.0.1:8080
export VERILAB_HTTP=:8090
export VERILAB_WEB_DIST="$(pwd)/verification-lab/web/dist"
export MYSQL_HOST=127.0.0.1:3306 MYSQL_USER=ticket MYSQL_PASSWORD=ticket MYSQL_DATABASE=ticketdb
export REDIS_HOST=127.0.0.1:6379
# Optional: VERILAB_REPO_ROOT=$(pwd) for gate subprocess tests

go run ./verification-lab/cmd/verilab
```

Open `http://localhost:8090`.

## Presets (`POST /api/runs`)

| Preset | Purpose |
|--------|---------|
| `smoke` | 100 bookings |
| `gates` / `phase-gates` | P1–P7 acceptance tests |
| `claims` | INV-1..7 scenarios |
| `benchmark-smoke` | 100 requests, all 200 |
| `benchmark-saturated-15k` | 15k unique users |
| `benchmark-exhaustion-mini` | 5k exhaustion attempts |
| `benchmark-exhaustion-15k` | 100k attempts (`confirm_heavy: true`) |
| `failure-lab` | Safe fault scenarios (`confirm_fault: true`) |
| `security` / `security-s7` | Slice 7 go-test probes (CORS, Redis AUTH, log redaction, prod profile) |
| `heavy-100k` | Load exhaustion (`confirm_heavy: true`) |

Export: `GET /api/runs/{id}/export`

Telemetry: `GET /api/stream` (SSE snapshots ~200ms)

## Make targets

```bash
make test-verilab      # go test ./verification-lab/...
make build-verilab-ui  # npm build
make run-verilab       # start lab (requires VERILAB_* env)
```

Golden benchmark exports: `./scripts/golden-evidence.sh` (requires running lab; see `docs/BENCHMARKS.md`).
