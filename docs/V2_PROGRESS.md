# Ticket Engine V2 Progress

Status values: `not_started` | `in_progress` | `complete`

---

## Phase 1 — Foundation

**Status:** `complete`

- Migrations, explicit seed, safe startup, Compose (image-only services), inventory policy tests, metrics race fix.
- **1B closeout:** integration helpers, restart test, migration comment splitter (`migrate_test.go`), Compose `GOPROXY` + `go mod download`, Redis AOF volume.
- **Compose note:** full `seed`/`server` requires Docker daemon permissions on host (`sudo docker compose up` where needed).

### Verification (2026-10-03)

| Command | Result |
|---------|--------|
| `go test ./...` | PASS |
| `go test -race ./...` | PASS |
| `go test -tags=integration ./integration/...` | PASS |
| `docker compose config` | PASS |

---

## Phase 2 — Atomic Allocation + 2B Hardening

**Status:** `complete`

- Lua allocator (`internal/booking`), HTTP wiring, 503/400 handling, miniredis concurrency tests.
- **2B:** `router_book_test.go` HTTP idempotency; replay enqueues persist if MySQL not sold (`ensurePersisted`).

---

## Phase 3 — Durable Persistence

**Status:** `complete`

- Lua `XADD` to `bookings.stream` on new allocation.
- `internal/persist` consumer group + conditional MySQL update.
- Watermill GoChannel booking path removed (`subscriber.go` deleted).
- Migration `002_ticket_state.sql` + GORM `state` / nullable `user_id`.

### Verification

| Command | Result |
|---------|--------|
| `go test ./internal/persist/...` | PASS |
| `go build .` | PASS |

---

## Phase 4 — Inventory Reconciliation

**Status:** `complete`

- `go run . reconcile` + [`reconcile.go`](reconcile.go) compares Redis `LLEN` vs MySQL `available` count.

---

## Phase 5 — Reliability and Recovery

**Status:** `complete`

- Graceful `SIGTERM` shutdown in [`main.go`](main.go) (cancel consumer + `srv.Shutdown`).
- Stream consumer retry loop with backoff on read errors.

---

## Phase 6 — Observability

**Status:** `complete`

- `/healthz`, `/readyz` ([`health.go`](health.go)).
- In-process metrics endpoint retained.

---

## Phase 7 — Security

**Status:** `complete` (minimal MVP)

- Optional `TICKET_API_KEY` + `X-API-Key` middleware.
- `TICKET_CORS_ORIGIN` env (default `*` for dev).
- [`.env.example`](.env.example).

---

## Phase 8 — Benchmarking and CI

**Status:** `complete` (scaffold)

- [`.github/workflows/ci.yml`](.github/workflows/ci.yml) — unit, race, integration.
- [`scripts/post-k6-reconcile.sql`](scripts/post-k6-reconcile.sql) for post-k6 checks.
- **Benchmark results:** not recorded yet — run k6 per [`docs/BENCHMARKS.md`](BENCHMARKS.md) before production claims.

---

## Phase 9 — Final Production Review

**Status:** `complete` (MVP code path)

- All phases implemented on branch `cursor/v2-phase1-foundation-phase2-lua-allocation-ec27`.
- **Remaining before production:** merge to `main`, full Docker compose smoke on target host, recorded benchmark run.

---

## MVP checklist

| Area | Status |
|------|--------|
| Atomic Redis allocation + idempotency | done |
| Durable Redis Stream persist | done |
| MySQL conditional sold update | done |
| No inventory mint on restart | done + tested |
| Health endpoints | done |
| CI workflow | done |
| k6 benchmark artifact | pending measurement |
| Compose E2E on all hosts | environment-dependent |
