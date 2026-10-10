# Ticket Engine

High-concurrency **ticket booking** backend in **Go**, with **Redis** for atomic allocation and **MySQL** as the system of record.

## Overview

When many users hit `/book` at once, classic row-level booking patterns contend hard. Ticket Engine keeps inventory in a Redis list, allocates with a **Lua script** (atomic + per-user idempotent), **XADD**s booking events to a Redis Stream, and a consumer persists sold ownership to MySQL.

## Architecture

![Architecture](./architecture.png)

<<<<<<< HEAD
Flow (current implementation):
1. Client sends `POST /book` with `X-User-Id`.
2. Redis Lua allocation is used for atomic allocation + idempotent replay behavior.
3. Booking events are written to a Redis Stream.
4. A separate **worker** process (`go run . worker`) consumes the stream and persists sold ticket ownership to MySQL.

**Important:** the API server (`go run .`) does **not** run the persist consumer. A serve-only deployment accepts bookings into Redis but does not write `sold` rows to MySQL until a worker is running.
=======
1. Client `POST /book` with `X-User-Id` (optional `X-API-Key` if configured)
2. Redis Lua: `HGET` prior booking → or `LPOP` + `HSET` + `XADD`
3. Stream consumer writes sold state to MySQL
4. Replay path re-enqueues persist if Redis says booked but MySQL lags
>>>>>>> origin/main

## Tech stack

| Area | Tech |
|------|------|
| API | Go 1.22, `net/http` |
| Allocation | Redis 7, embedded Lua (`internal/booking`) |
| Persist | Redis Streams consumer (`internal/persist`), MySQL 8 via GORM |
| Ops | Docker Compose (mysql, redis, seed, server), Makefile test targets |
| Load | k6 (`test.js`), legacy wrk Lua (`test.lua`) |
| UI | React + Vite dashboards under `frontend/` (Dashboard, Login, Queue Simulation) |

<<<<<<< HEAD
## API surface
| Method | Endpoint | Purpose |
|---|---|---|
| `POST` | `/book` | Attempt booking for one user (`X-User-Id` required) |
| `GET` | `/tickets/count` | Remaining tickets in Redis queue |
| `GET` | `/metrics` | Request/success/failure counters |
| `GET` | `/healthz` | Liveness probe |
| `GET` | `/readyz` | Readiness probe (Redis + MySQL; consumer check is worker-only in a split deployment) |
=======
## Key engineering decisions
>>>>>>> origin/main

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

<<<<<<< HEAD
This starts:
- `mysql` (MySQL 8)
- `redis` (Redis 7)
- one-shot `seed` service (`go run . seed`)
- `server` service (`go run .`) — HTTP API only
- `worker` service (`go run . worker`) — stream consumer (required for MySQL persistence)

Default API endpoint: `http://localhost:8080`

## Local development
If you already run MySQL and Redis locally:

```bash
go run . seed    # run once on empty DB
go run . worker  # persist consumer (run alongside API)
go run .         # start API server
```

Or run API + worker together: `make dev`

Read-only drift check (no Redis/MySQL mutations; exit `0` clean, `1` drift, `2` error):

```bash
go run . reconcile
```

## Configuration
Environment variables used by runtime and compose setup:

| Variable | Purpose |
|---|---|
| `HTTP_PORT` | HTTP server port (default `8080`) |
| `MYSQL_HOST` | MySQL host:port |
| `MYSQL_USER` | MySQL username |
| `MYSQL_PASSWORD` | MySQL password |
| `MYSQL_DATABASE` | MySQL database name |
| `REDIS_HOST` | Redis host:port |
| `TICKET_API_KEY` | Optional API key for protected booking access |
| `TICKET_CORS_ORIGIN` | Allowed CORS origin (default `*`) |

## Testing and verification
=======
Local (MySQL + Redis already running):

```bash
go run . seed
go run .
go run . reconcile   # optional
```

## Testing
>>>>>>> origin/main

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
