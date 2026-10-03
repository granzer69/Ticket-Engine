# Benchmarks Plan

**No results are recorded here until measured on a tagged commit.**

## Goals

Compare V1 and V2 under the same hardware, seed, and inventory (15,000 tickets).

## Load profiles

| Profile | VUs / rate | Duration | Purpose |
|---------|------------|----------|---------|
| Baseline | 100, 1k, 5k, 10k | 60s | Concurrency sweep |
| Target RPS | constant 500, 3k, 5k req/s | 3m | Throughput |
| Sell-out | 3k req/s | until 409/404 dominate | Boundary |
| Retry | 20% same-user replay | 3m | Idempotency (V2) |
| Crash | 3k req/s, kill API @30s | 3m | Recovery (V2) |
| Soak | 1k req/s | 45m | Leaks |

Tooling: `test.js` (k6); replace legacy `test.lua` wrk script for correctness runs.

## Metrics

- Throughput (successful bookings/s)
- Latency P50, P95, P99
- Error rate by status (409 sold out ≠ error)
- CPU, memory (host + containers)
- Redis command latency (INFO/latency doctor)
- MySQL slow query / commits per second
- Worker lag (stream pending — V2)

## Correctness checks (post-run + after quiesce)

1. Distinct `ticket_id` in 200 responses = successful unique users.
2. No ticket_id mapped to two users.
3. No user with two different ticket_ids (retries must repeat same id — V2).
4. `COUNT(sold) <= 15000` (seed cap).
5. `mysql_sold + redis_available (+ allocated V2) = 15000` after quiesce.
6. Crash profile: equalities hold after restart without new ids.

## Recording results

When run, append:

```markdown
## Run YYYY-MM-DD
- Git SHA:
- Profile:
- RPS / P95:
- Correctness: PASS/FAIL
- Notes:
```
