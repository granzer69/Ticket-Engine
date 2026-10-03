# Failure Matrix

Legend: **V1** = current behavior; **Target** = V2 end state; **Phase** = when we intend to fix.

| Failure | Detection | V1 behavior | Target behavior | Recovery | Test | Phase |
|---------|-----------|-------------|-----------------|----------|------|-------|
| Redis unavailable | `LPOP`/`HINCRBY` error | Often 404/429 | 503, no mutation | Retry client | Integration Redis down | 2+ |
| MySQL unavailable | Worker `UPDATE` error | HTTP 200 already sent | Allocated + stream pending | Worker retry | Pause MySQL | 3+ |
| Worker crash mid-handler | Stream pending / lost msg | GoChannel message lost | Pending entry reclaimed | `XAUTOCLAIM` | Kill mid-worker | 3+ |
| API crash after 200 | Client got ticket id | MySQL may be stale | Stream retains job | Consumer on restart | Kill API | 3+ |
| Process restart | Boot logs | Could mint rows (fixed P1) | No mint; reconcile | Seed/sync policy | Restart test | 1/4 |
| Duplicate request | Same user id | 200 same ticket (Phase 2) | 200 same ticket | Lua `HGET` replay | `internal/booking` tests + k6 retry | 2 done |
| Request timeout | Client retry | Ambiguous | Idempotent 200 | Same as duplicate | k6 | 2 |
| Redis ok, persist fails | Worker error | Silent gap | Retry + DLQ | Ops / DLQ | Integration | 3+ |
| Persist ok, response lost | Client retry | 429 | 200 same ticket | Idempotency | Integration | 2 |
| Duplicate worker delivery | Two updates | Possible race log | Conditional UPDATE | DB constraint | Integration | 3 |
| Partial shutdown | SIGTERM | Abrupt | Drain consumers | Graceful shutdown | Manual | 5 |

## Phase 1 scope

- **Process restart / inventory:** server start must not increase row count; documented in `seed.go`.
- **Redis sync:** optional rebuild does not `DEL hash:user`.

Tests: `seed_policy_test.go`, `integration/seed_test.go` (when services up).
