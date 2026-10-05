# Ticket Engine

A high-concurrency ticket booking backend built with Go, Redis, and MySQL.

## Table of Contents
- [Why this project](#why-this-project)
- [Current architecture](#current-architecture)
- [Key capabilities](#key-capabilities)
- [API surface](#api-surface)
- [Quick start (Docker)](#quick-start-docker)
- [Local development](#local-development)
- [Configuration](#configuration)
- [Testing and verification](#testing-and-verification)
- [Load testing and reconciliation](#load-testing-and-reconciliation)
- [V2 roadmap and docs](#v2-roadmap-and-docs)

## Why this project
Traditional databases struggle when very large numbers of users attempt to book at the same time. Ticket Engine moves allocation pressure to Redis, keeps booking allocation atomic, and persists booking state asynchronously to MySQL.

## Current architecture
![Architecture](./architecture.png)

Flow (current implementation):
1. Client sends `POST /book` with `X-User-Id`.
2. Redis Lua allocation is used for atomic allocation + idempotent replay behavior.
3. Booking events are written to a Redis Stream.
4. A separate **worker** process (`go run . worker`) consumes the stream and persists sold ticket ownership to MySQL.

**Important:** the API server (`go run .`) does **not** run the persist consumer. A serve-only deployment accepts bookings into Redis but does not write `sold` rows to MySQL until a worker is running.

This design gives fast allocation while preserving eventual consistency in the system of record.

## Key capabilities
- **Atomic booking allocation** in Redis to prevent duplicate ticket assignment.
- **Idempotent user booking behavior** so repeated requests return consistent outcomes.
- **Asynchronous persistence** using Redis Streams + consumer processing.
- **Operational endpoints** for health, readiness, counters, and ticket inventory visibility.
- **Explicit inventory seeding** via command (`go run . seed`) instead of startup auto-minting.

## API surface
| Method | Endpoint | Purpose |
|---|---|---|
| `POST` | `/book` | Attempt booking for one user (`X-User-Id` required) |
| `GET` | `/tickets/count` | Remaining tickets in Redis queue |
| `GET` | `/metrics` | Request/success/failure counters |
| `GET` | `/healthz` | Liveness probe |
| `GET` | `/readyz` | Readiness probe (Redis + MySQL; consumer check is worker-only in a split deployment) |

Example booking request:

```bash
curl -X POST http://localhost:8080/book \
  -H "X-User-Id: 101"
```

## Quick start (Docker)

```bash
docker compose up
```

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

Useful command:

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

```bash
make test
make test-race
make test-integration   # requires MySQL + Redis
```

## Load testing and reconciliation
- k6 script: [`test.js`](test.js)
- post-run SQL reconcile script: [`scripts/post-k6-reconcile.sql`](scripts/post-k6-reconcile.sql)
- legacy V1 script: [`test.lua`](test.lua)

## V2 roadmap and docs
- V2 target spec: [`docs/V2_SPEC.md`](docs/V2_SPEC.md)
- V2 phase progress: [`docs/V2_PROGRESS.md`](docs/V2_PROGRESS.md)
- failure/recovery matrix: [`docs/FAILURE_MATRIX.md`](docs/FAILURE_MATRIX.md)
- benchmark plan: [`docs/BENCHMARKS.md`](docs/BENCHMARKS.md)

Current phase status is tracked in `docs/V2_PROGRESS.md`.
