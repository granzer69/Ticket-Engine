# Ticket Engine

High-concurrency **ticket booking** backend in **Go**, with **Redis** for atomic allocation and **MySQL** as the system of record.

## Overview

When many users hit `/book` at once, classic row-level booking patterns contend hard. Ticket Engine keeps inventory in a Redis list, allocates with a **Lua script** (atomic + per-user idempotent), **XADD**s booking events to a Redis Stream, and a consumer persists sold ownership to MySQL.

## Architecture

![Architecture](./architecture.png)

1. Client `POST /book` with `X-User-Id` (optional `X-API-Key` if configured)
2. Redis Lua: `HGET` prior booking → or `LPOP` + `HSET` + `XADD`
3. Stream consumer writes sold state to MySQL
4. Replay path re-enqueues persist if Redis says booked but MySQL lags

## Tech stack

| Area | Tech |
|------|------|
| API | Go 1.22, `net/http` |
| Allocation | Redis 7, embedded Lua (`internal/booking`) |
| Persist | Redis Streams consumer (`internal/persist`), MySQL 8 via GORM |
| Ops | Docker Compose (mysql, redis, seed, server), Makefile test targets |
| Load | k6 (`test.js`), legacy wrk Lua (`test.lua`) |
| UI | React + Vite dashboards under `frontend/` (Dashboard, Login, Queue Simulation) |

## Key engineering decisions

- **Redis owns hot allocation**; MySQL owns durable sold state (eventual consistency by design)
- **Idempotent user booking**: same `user_id` retries return the same `ticket_id` with HTTP 200
- **Explicit seed**: `go run . seed` mints inventory (default 15,000); server start does not auto-mint
- **Race-aware metrics**: counters protected for concurrent `/metrics` under load
- **V2 phased roadmap** documented under `docs/` — phases 1–2 complete; later phases (full recovery/observability/security/CI bench gate) tracked in `docs/V2_PROGRESS.md`

## Features

- Atomic booking allocation (no double-assign of the same ticket id under concurrent clients — covered by unit/integration tests)
- Idempotent replay behavior
- Async MySQL persistence via stream consumer
- Health / readiness / inventory count / metrics endpoints
- Docker Compose local stack
- k6 staged load profile (ramp toward multi-thousand concurrent VUs — **results not yet recorded** in `docs/BENCHMARKS.md`)

## API

| Method | Path | Purpose |
|--------|------|---------|
| POST | `/book` | Book one ticket for `X-User-Id` |
| GET | `/tickets/count` | Remaining Redis queue length |
| GET | `/metrics` | Request / success / failure counters |
| GET | `/healthz` | Liveness |
| GET | `/readyz` | Readiness (Redis + MySQL + consumer checks) |

```bash
curl -X POST http://localhost:8080/book -H "X-User-Id: 101"
```

## Performance / results

**No published throughput or latency numbers yet.** `docs/BENCHMARKS.md` is a measurement plan only — append runs with git SHA when executed. The repo includes k6 scripts and correctness checks (unique ticket↔user mapping, inventory conservation) to support that work.

Do **not** treat the old GitHub description claim of “100k requests” as a verified result.

## Getting started

```bash
docker compose up
# API: http://localhost:8080
```

Local (MySQL + Redis already running):

```bash
go run . seed
go run .
go run . reconcile   # optional
```

## Testing

```bash
make test
make test-race
make test-integration   # needs MySQL + Redis
```

## Project structure

```text
internal/booking/   Lua allocator
internal/persist/   stream consumer
migrations/         SQL schema
docs/               V2 spec, progress, failure matrix, benchmark plan
frontend/           React/Vite UI experiments
test.js             k6 load script
```

## Future improvements

See `docs/V2_PROGRESS.md` for remaining phases: reconciliation rules, DLQ/backoff, richer observability, security hardening, CI benchmark gate, and recorded load results.
