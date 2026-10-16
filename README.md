# Ticket Engine

High-concurrency **ticket booking** backend in **Go**, with **Redis** for atomic allocation and **MySQL** as the system of record.

## Overview

When many users hit `/book` at once, classic row-level booking patterns contend hard. Ticket Engine keeps inventory in a Redis list, allocates with a **Lua script** (atomic + per-user idempotent), **XADD**s booking events to a Redis Stream, and a consumer persists sold ownership to MySQL.

## Architecture

![Architecture](./architecture.png)

Flow (current implementation):
1. Client sends `POST /book` with `X-User-Id` (optional `X-API-Key` if configured).
2. Redis Lua allocation: `HGET` prior booking, or `LPOP` + `HSET` + `XADD` (atomic + idempotent replay).
3. Booking events go to a Redis Stream.
4. A separate **worker** process (`go run . worker`) consumes the stream and persists sold ticket ownership to MySQL. Replay re-enqueues persist if Redis says booked but MySQL lags.

**Important:** the API server (`go run .`) does **not** run the persist consumer. A serve-only deployment accepts bookings into Redis but does not write `sold` rows to MySQL until a worker is running.

## Tech stack

| Area | Tech |
|------|------|
| API | Go 1.22, `net/http` |
| Allocation | Redis 7, embedded Lua (`internal/booking`) |
| Persist | Redis Streams consumer (`internal/persist`), MySQL 8 via GORM |
| Ops | Docker Compose (mysql, redis, seed, server, worker), Makefile test targets |
| Load | k6 (`test.js`), legacy wrk Lua (`test.lua`) |
| UI | **Verification Lab** (`verification-lab/`) — live telemetry console; legacy demos under `frontend/` |

## Key engineering decisions

- **Redis owns hot allocation**; MySQL owns durable sold state (eventual consistency by design)
- **Idempotent user booking**: same `user_id` retries return the same `ticket_id` with HTTP 200
- **Explicit seed**: `go run . seed` mints inventory (default 15,000); server start does not auto-mint
- **Race-aware metrics**: counters protected for concurrent `/metrics` under load
- **V3** worker split, stream consumer, PEL reclaim, DLQ, read-only reconcile, verification lab — see `docs/V2_PROGRESS.md` and `docs/VERIFICATION_LAB.md`

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
| GET | `/readyz` | Readiness (Redis + MySQL; consumer check is worker-only in a split deployment) |

```bash
curl -X POST http://localhost:8080/book -H "X-User-Id: 101"
```

## Performance / results

**Recorded lab benchmarks** (smoke, exhaustion, gates) are in [`docs/BENCHMARKS.md`](docs/BENCHMARKS.md) with git SHA and honest caveats (15k seed vs 100k attempts). k6 (`test.js`) cross-checks via nightly workflow.

Do **not** treat any old GitHub description claim of unverified request volume as a measured result.

## Getting started

```bash
docker compose up
# API: http://localhost:8080
```

This starts:
- `mysql` (MySQL 8)
- `redis` (Redis 7)
- one-shot `seed` service (`go run . seed`)
- `server` service (`go run .`) — HTTP API only
- `worker` service (`go run . worker`) — stream consumer (required for MySQL persistence)

Optional Compose profiles:

- `docker compose --profile multi-worker up` — adds `worker-b` (second consumer name) for P4-style tests
- `docker compose --profile lab up` — adds `verilab` on port 8090 (builds UI on first start)
- Set `REDIS_PASSWORD` in `.env` to enable Redis AUTH for all app services

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

| Variable | Purpose |
|---|---|
| `HTTP_PORT` | HTTP server port (default `8080`) |
| `MYSQL_HOST` | MySQL host:port |
| `MYSQL_USER` | MySQL username |
| `MYSQL_PASSWORD` | MySQL password |
| `MYSQL_DATABASE` | MySQL database name |
| `REDIS_HOST` | Redis host:port |
| `REDIS_PASSWORD` | Redis AUTH password when enabled |
| `TICKET_API_KEY` | Optional API key for protected booking access |
| `TICKET_CORS_ORIGIN` | Allowed CORS origin (default `*`) |
| `TICKET_ENV` | Set to `production` for fail-fast API key + non-wildcard CORS |
| `TICKET_SEED_COUNT` | Dev-only seed row count (default 15000; never auto-mint on serve) |
| `TICKET_STREAM_MAXLEN` | Approximate `MAXLEN ~` on bookings stream XADD (optional backpressure) |

## Verification Lab

Observability and proof console (gates P1–P7, benchmarks, failure lab). Requires API + worker + Redis + MySQL.

```bash
make build-verilab-ui
make run-verilab   # set VERILAB_TARGET, MYSQL_*, REDIS_HOST — see docs/VERIFICATION_LAB.md
```

## Testing

```bash
make test
make test-race
make test-integration   # needs MySQL + Redis
make test-verilab
```

## Project structure

```text
internal/booking/   Lua allocator
internal/persist/   stream consumer
migrations/         SQL schema
docs/               V2 spec, progress, VERIFICATION_LAB.md, benchmark plan
verification-lab/   verilab control plane + React console
frontend/           Legacy React demos (not wired to lab telemetry)
test.js             k6 load script
```

## Future improvements

See `docs/V2_PROGRESS.md` for phase status; `docs/V3_SECURITY_PROOF.md` for Slice 7 evidence.
