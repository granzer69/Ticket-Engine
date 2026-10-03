# Architecture

## Current (V1) — as implemented

```
                    POST /book
                         |
                         v
              +---------------------+
              |  Go HTTP server     |
              |  (single process)   |
              +----------+----------+
                         |
         +---------------+---------------+
         |                               |
         v                               v
  +-------------+               +------------------+
  | Redis       |               | Watermill        |
  | LIST tickets|               | GoChannel        |
  | HASH hash:  |               | ticket.update    |
  |     user    |               +--------+---------+
  +-------------+                        |
         ^                               v
         |                      +--------+---------+
         |                      | updateTicket()   |
         |                      | (same process)   |
         |                      +--------+---------+
         |                               v
         |                      +------------------+
         +-- sync on startup ---| MySQL tickets    |
                                +------------------+
```

### Request flow

1. Client sends `X-User-Id` header.
2. Redis Lua: idempotent `HGET` on `hash:user`, else `LPOP` + `HSET` (Phase 2).
3. Message published to in-memory channel.
4. JSON 200 returned.
5. Handler updates MySQL asynchronously.

### Consistency boundary

- **Fast path truth:** Redis list + user hash (until pop).
- **Durable truth for sold tickets:** MySQL after worker commit.
- **Gap:** HTTP 200 can precede MySQL; crash can lose in-memory messages.

### Idempotency (Phase 2)

Same `X-User-Id` retry returns HTTP 200 with the same `ticket_id` (Redis Lua). New allocations publish to GoChannel once; replays do not republish.

### Startup (after Phase 1)

- Migrations from `migrations/`.
- **No** automatic MySQL insert on serve.
- `go run . seed` creates 15k rows only when table empty.
- Redis: if queue empty, load available ids from MySQL; optional `TICKET_ENGINE_REDIS_SYNC=1` rebuild without clearing `hash:user`.

---

## Target (V2)

```
Client
  |
  v
Go API
  |
  v
Redis EVALSHA (allocate + enqueue)
  |
  v
Redis Stream (bookings)
  |
  v
Consumer group -> MySQL conditional UPDATE
```

### Failure recovery (target)

Pending stream entries reclaimed after worker crash; conditional MySQL update makes redelivery safe.

See `docs/FAILURE_MATRIX.md`.
