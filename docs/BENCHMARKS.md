# Benchmarks

Recorded runs use the **Verification Lab** presets against a **15k** seed (`go run . seed`). Exports are archived under the project store at `internal/golden-benchmarks/`.

**Environment (2026-10-16):** single VM, local MySQL 8 + Redis 7, API + worker on host, `verilab` on `:8090`. Client RPS/latency are from the lab load generator (includes HTTP RTT; not production SLA).

| Profile | Git SHA | Result | RPS (approx) | P95 (ms) | Notes |
|---------|---------|--------|--------------|----------|-------|
| `benchmark-smoke` | `f679d1a`+ | PASS | 12.5k | 1 | 100/100 HTTP 200; `queue_consumed` 100 |
| `benchmark-saturated-15k` | `f679d1a`+ | FAIL* | 23.5k | 2 | *Chained after smoke on same seed: 14 900/15 000 HTTP 200 (100 tickets already consumed). Re-run alone requires empty DB + Redis before seed. |
| `benchmark-exhaustion-mini` | `f679d1a`+ | PASS | (see export) | — | 5k exhaustion mode; pass uses queue rules |
| `benchmark-exhaustion-15k` | `f679d1a`+ | PASS | 26.4k | 2 | 100k logical attempts vs 15k inventory |
| `gates` | `f679d1a`+ | PASS | — | — | P1–P7; **P6** reconcile CLI integration PASS |
| `failure-lab` | `f679d1a`+ | PASS | — | — | `confirm_fault`; see export for scenario rows |

## Run 2026-10-16

- **Host:** Cloud agent VM (Go 1.22), fresh `go run . seed` then lab presets in order (smoke → saturated → mini → exhaustion-15k → gates → failure-lab).
- **Correctness:** Smoke and exhaustion profiles passed lab pass rules; saturated failed only when inventory was already partially consumed.
- **Artifacts:** JSON exports `2026-10-16-*-f679d1a.json` in project store `internal/golden-benchmarks/`.

### Reproduce

```bash
# Stack
go run . seed
go run . worker &
go run .
make build-verilab-ui && make run-verilab

# Evidence script (one export per preset)
GOLDEN_ARCHIVE=/path/to/archive ./scripts/golden-evidence.sh
```

For a **clean** `benchmark-saturated-15k`, reset MySQL tickets + `FLUSHALL` on Redis, seed once, run **only** that preset (do not run smoke first on the same seed).

## Legacy k6 plan

Older k6 profiles (`test.js`) remain documented for cross-check; nightly workflow `.github/workflows/k6-nightly.yml` runs a short k6 smoke against CI services.
