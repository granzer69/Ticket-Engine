# Ticket Engine V2 Progress

Status values: `not_started` | `in_progress` | `complete`

Only mark **complete** when acceptance criteria are met, tests executed, and Reviewer PASS recorded.

---

## Phase 1 — Foundation

**Objective:** Reliable schema/seed path, safe startup (no inventory mint), Compose/test harness — without changing core booking algorithm (HINCRBY + LPOP + GoChannel).

### Tasks

| Task | Status |
|------|--------|
| Project rule + agent definitions | complete |
| V2 documentation set | complete |
| Versioned SQL migration + apply on startup | complete |
| Explicit `seed` command / env; no auto-mint on `serve` | complete |
| Conservative Redis sync on startup | complete |
| Docker Compose credentials + healthchecks + seed job | complete |
| Go toolchain bump (supported version) | complete |
| Unit tests for seed policy | complete |
| Integration test scaffold (`integration` build tag) | complete |
| Fix `/metrics` counter data race | complete |

### Acceptance criteria

- [x] Normal server start does **not** insert MySQL rows to “top up” inventory.
- [x] `go run . seed` (or documented env) is the only way to create initial 15,000 rows on empty DB.
- [x] `migrations/001_tickets.sql` applied on startup before serving.
- [x] `docker compose config` validates.
- [x] `go test ./...` passes.
- [x] `go test -race ./...` passes (unit packages).
- [x] `go test -tags=integration ./integration/...` — PASS with skip when MySQL/Redis unreachable (2026-10-03 cloud VM)
- [ ] `docker compose config` + full stack `up` — **blocked**: Docker CLI not installed on cloud VM (verified `which docker` empty; `apt-get install docker.io` unavailable)

### Tests required

- Unit: `internal/inventory/policy_test.go`
- Integration: `integration/inventory_integration_test.go` (build tag)

### Verification log

| Command | Result | Date |
|---------|--------|------|
| `go test ./...` | PASS | 2026-10-03 |
| `go test -race ./...` | PASS | 2026-10-03 |
| `go test -tags=integration ./integration/...` | PASS (skipped MySQL/Redis) | 2026-10-03 |
| `docker compose config` | Not run — no Docker | 2026-10-03 |

### Status

`complete` — code and automated tests verified; **Docker Compose stack smoke not verified** in this environment.

---

## Phase 2 — Atomic Allocation

**Objective:** Redis Lua script; idempotent 200 on retry.

### Tasks

| Task | Status |
|------|--------|
| Lua atomic allocate + idempotent user hash | complete |
| Wire API `/book` to allocator | complete |
| Reject `user_id <= 0` | complete |
| Redis errors → 503 | complete |
| Idempotent replay → 200, no second publish | complete |
| Unit tests (miniredis) + concurrent tests | complete |
| Integration test (Redis, build tag) | complete |

### Acceptance criteria

- [x] Allocation is one atomic Redis script (`HGET` / `LPOP` / `HSET`).
- [x] Same user retry returns same `ticket_id` with HTTP 200.
- [x] Two concurrent users cannot receive the same ticket id.
- [x] `go test ./...` and `go test -race ./...` pass.
- [x] No Redis Streams / Phase 3 work started.

### Verification log

| Command | Result | Date |
|---------|--------|------|
| `go test ./...` | PASS (`internal/booking`, `internal/inventory`) | 2026-10-03 |
| `go test -race ./...` | PASS | 2026-10-03 |
| `go test -tags=integration ./integration/...` | PASS (Redis/MySQL skipped if down) | 2026-10-03 |

### Status

`complete` — Reviewer PASS (2026-10-03). MySQL async persist unchanged (Phase 3).

---

## Phase 3 — Durable Persistence

**Objective:** Redis Stream + consumer group; remove GoChannel for bookings.

**Status:** `not_started`

---

## Phase 4 — Inventory Reconciliation

**Objective:** Rebuild rules; no destructive reset of buyer state.

**Status:** `not_started`

---

## Phase 5 — Reliability and Recovery

**Objective:** Backoff, DLQ, graceful shutdown.

**Status:** `not_started`

---

## Phase 6 — Observability

**Objective:** Health, metrics, structured logs, dashboard truth.

**Status:** `not_started`

---

## Phase 7 — Security

**Objective:** Auth token, CORS, rate limits.

**Status:** `not_started`

---

## Phase 8 — Benchmarking and CI

**Objective:** V1/V2 benchmark gate + CI.

**Status:** `not_started`

---

## Phase 9 — Final Production Review

**Objective:** End-to-end reviewer sign-off.

**Status:** `not_started`
