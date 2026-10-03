# Ticket Engine V2 Specification

## 1. V2 objective

Turn the existing high-concurrency ticket booking prototype into a production-oriented system where **correctness, concurrency safety, and recoverability** are provable before performance claims.

Incremental delivery: document the target architecture, implement in phases (`docs/V2_PROGRESS.md`), and do not rewrite working paths without justification.

## 2. Current V1 architecture (verified in repository)

Single Go process (`main.go`, `router.go`, `init.go`, `subscriber.go`, `model.go`):

- HTTP `POST /book`, `GET /tickets/count`, `GET /metrics`
- Redis: list `tickets` (available ids), hash `hash:user` (per-user booking counter)
- Watermill **GoChannel** (in-process, not durable) topic `ticket.update`
- MySQL `tickets` table via GORM `AutoMigrate`

Frontend: `frontend/Dashboard` calls the API; other frontend bundles are mostly UI demos.

## 3. Current booking lifecycle (verified)

1. `HINCRBY hash:user <uid> 1` — proceed only if result is `1`
2. `LPOP tickets` — atomic pop of ticket id
3. Publish JSON ticket to GoChannel
4. HTTP 200 to client
5. Worker `updateTicket`: `SELECT user_id`, then `UPDATE` if still `user_id = 0` (dummy unsold sentinel)

There is **no** explicit `reserved` state in Redis or MySQL.

## 4. V1 problems verified from the repository

| Issue | Verified |
|-------|----------|
| HTTP success before MySQL commit | Yes — `router.go` returns 200 after publish; worker is async |
| In-memory Watermill GoChannel | Yes — `gochannel.NewGoChannel` in `init.go` |
| Startup can mint MySQL inventory | Yes — `prepareData` inserted rows when `user_id=0` count &lt; 15000 (fixed in Phase 1) |
| Startup could clear `hash:user` | Yes — on Redis reload (Phase 1 reduces destructive sync) |
| Weak idempotency | Yes — retry returns 429 after `HINCRBY` |
| No Go `_test.go` files | Yes (addressed in Phase 1) |
| `test.lua` uses `GET /ticket` | Yes — not `POST /book` |
| Compose DB credentials mismatch | Yes — `ming`/`test` vs MySQL service `ticketdb` only (fixed in Phase 1) |
| Benchmark claims not proven in repo | Yes — no committed results; k6 thresholds differ from stated P95 |

**Not verified in this run:** historical 10K users / 3K rps / P95 &lt;250ms in production — no artifacts in tree.

## 5. Target V2 architecture

```
Client
  ↓
Go API
  ↓
Atomic Redis allocation (Lua — Phase 2+)
  ↓
Durable Redis Stream (Phase 3+)
  ↓
Consumer group worker(s)
  ↓
MySQL (source of truth for completed sales)
```

Implement incrementally; see `docs/V2_PROGRESS.md`.

## 6. Target booking lifecycle

1. Idempotent allocate in Redis (one round trip)
2. Durable enqueue (stream entry in same atomic unit as allocation — later phase)
3. HTTP 200 means **allocated and enqueued**, not necessarily MySQL `sold`
4. Worker conditional `UPDATE` with ack after commit
5. `GET` booking status: MySQL first, Redis allocation second (later phase)

## 7. Ticket state model (target)

| State | Meaning |
|-------|---------|
| `available` | In Redis list / MySQL `user_id = 0` (V1 sentinel; may become NULL + `state` column later) |
| `allocated` | Removed from list; owner recorded in Redis; persist job pending |
| `sold` | MySQL row updated; durable owner |

**Unresolved:** whether to replace `user_id = 0` sentinel with `NULL` + `state` enum (planned before Phase 3).

## 8. Redis responsibilities (target)

- Atomic allocation and idempotency keys
- Durable stream of persist jobs (Phase 3+)
- Optional owner/index keys for reconciliation (Phase 4+)
- Not the long-term sole source of truth for completed sales

## 9. MySQL responsibilities (target)

- Authoritative record of **sold** tickets
- Constraints: unique `user_id` for sold rows, inventory cap tied to seed
- Versioned schema via SQL migrations (`migrations/`)

## 10. Idempotency strategy (target)

- User id (or explicit `Idempotency-Key`) as idempotency key
- Retry returns same `ticket_id` with 200
- Worker redelivery safe via conditional update + unique constraints

**Phase 2 (implemented):** Redis Lua script on `hash:user` stores `user_id → ticket_id`; retry returns the same ticket with HTTP 200 without a second `LPOP`. Async MySQL persist unchanged until Phase 3.

## 11. Failure/recovery strategy (target)

See `docs/FAILURE_MATRIX.md`. Phase 1 improves **startup inventory** and **schema path** only.

## 12. Booking invariants

Listed in `.cursor/rules/ticket-engine-v2.mdc`.

## 13. Testing strategy

- Unit tests for seed/sync policy (Phase 1)
- Integration tests with `//go:build integration` against Docker MySQL/Redis (Phase 1+)
- Race detector on packages with concurrency
- k6 + post-run SQL reconciliation (`docs/BENCHMARKS.md`) — Phase 8

## 14. Benchmark strategy

`docs/BENCHMARKS.md` — no results claimed until measured and recorded.

## 15. Security requirements (target)

- Server-issued identity (replace raw `X-User-Id`) — Phase 7
- CORS restriction, secrets from env, rate limiting — Phase 7
- Phase 1: align Compose secrets; no new auth

## 16. Observability requirements (target)

- Health/readiness, structured logs, Prometheus — Phase 6
- Phase 1: fix metrics counter read race if touched

## 17. Production-readiness criteria

Phases 1–9 in `docs/V2_PROGRESS.md`; production gate includes invariant tests, benchmark reconciliation, and reviewer PASS per phase.
