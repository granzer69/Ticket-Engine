# V3 Security proof (Slice 7)

This document records **reproducible evidence** for security claims. A claim is **proven** only when it has an automated test (or CI step), an executed result, and a reference to that CI run.

**Status:** Slice 7 not complete — sections below are **planned** until implementation and CI runs land.

## How to use

For each claim:

1. Implement behavior (see `internal/v3-slice7-claims-matrix.md` in the project agent store, or the Slice 7 PR description).
2. Add or extend automated tests.
3. Run locally, then merge and capture the GitHub Actions **workflow run URL** for `go test ./...` and integration (if applicable).
4. Fill **Actual result** and **CI evidence** — do not invent run IDs.

## Claims index

| ID | Claim | Test package / command | Evidence |
|----|-------|------------------------|----------|
| S7-API-1 | API key required when configured | `go test -run TestAPIKey -count=1 ./...` | Local PASS (claim-tests branch); CI pending |
| S7-CORS-1 | CORS allows only configured origin | `go test -run TestCORS -count=1 ./...` | *Pending* |
| S7-REDIS-1 | Redis AUTH when configured | `go test -tags=integration -run TestRedisAuth ./integration/...` | *Pending* |
| S7-LOG-1 | Logs omit secrets and raw user ids | `go test -run TestLogRedaction -count=1 ./...` | *Pending* |
| S7-SEC-1 | No hardcoded production secrets | CI GitGuardian + `go test -run TestSecrets -count=1 ./...` | GitGuardian: see PR checks; custom test *Pending* |
| S7-PROD-1 | Production profile fails without required config | `go test -run TestProductionConfig -count=1 ./...` | *Pending* |
| S7-RATE-1 | Rate limit on `/book` (if in scope) | `go test -run TestRateLimit -count=1 ./...` | *Pending* |

---

## S7-API-1 — API authentication

**Claim:** When `TICKET_API_KEY` is set, unauthenticated or invalid clients cannot complete protected booking.

**Threat/condition:** Anonymous or wrong-key callers must not reach allocation.

**Test procedure:**

- `go test -count=1 -run 'TestAPIKey' ./...`
- Cases: no `X-API-Key`; wrong key; correct key.

**Expected result:**

- Missing key → HTTP 401, no booking side effects.
- Invalid key → HTTP 401.
- Valid key → HTTP 200 (with valid `X-User-Id` and inventory).

**Actual result (local, 2026-10-05):** `go test -count=1 -run 'TestAPIKey' ./...` — PASS (`ticketengine`, `internal/security`). Missing and invalid `X-API-Key` return HTTP 401 with body `Unauthorized`; valid key allows booking; env unset keeps `/book` open.

**CI evidence:** Pending CI verification

**Known limitations:** When `TICKET_API_KEY` is unset, `/book` remains open (dev default). API key does not authenticate `X-User-Id` (application identifier only).

---

## S7-CORS-1 — Cross-origin access

**Claim:** With production-style configuration, only explicitly allowed origins receive permissive CORS headers.

**Threat/condition:** Browser clients on hostile origins must not treat the API as credentialed same-site.

**Test procedure:**

- `go test -count=1 -run 'TestCORS' ./...`
- Allowed origin, disallowed origin, OPTIONS preflight.

**Expected result:**

- Allowed → `Access-Control-Allow-Origin` matches config; POST succeeds.
- Disallowed → origin header not reflected (or request blocked per policy).
- Preflight → 200 with expected `Access-Control-Allow-Methods` / headers.

**Actual result:** *Not recorded yet.*

**CI evidence:** *Pending.*

**Known limitations:** Default `TICKET_CORS_ORIGIN=*` is dev-only; document production value.

---

## S7-REDIS-1 — Redis authentication

**Claim:** Application supports password-protected Redis and fails safely when credentials are wrong or required but missing (production profile).

**Threat/condition:** Unauthenticated Redis on a shared network.

**Test procedure:**

- Unit: miniredis with `AUTH` or integration against Redis with `--requirepass`.
- `go test -tags=integration -count=1 -run 'TestRedisAuth' ./integration/...`

**Expected result:**

- Valid password → ping OK, booking path works.
- Invalid password → connection error at startup or first use; no silent fallback.
- Production profile without password env → non-zero exit on startup.

**Actual result:** *Not recorded yet.*

**CI evidence:** *Pending.*

**Known limitations:** Current code uses `REDIS_HOST` only.

---

## S7-LOG-1 — Logging redaction

**Claim:** Production logs do not contain API keys, authorization headers, database/redis passwords, or raw user identifiers (per Slice 7 policy).

**Threat/condition:** Credential or PII leakage via log aggregation.

**Test procedure:**

- `go test -count=1 -run 'TestLogRedaction' ./...`
- Capture `log` output (or injected logger) while exercising `/book` with known key and user id.

**Expected result:** Captured logs contain none of the injected secret substrings.

**Actual result:** *Not recorded yet.*

**CI evidence:** *Pending.*

**Known limitations:** Third-party driver logs excluded unless documented.

---

## S7-SEC-1 — Secrets not in source

**Claim:** Production secrets are supplied via configuration, not committed in application source.

**Test procedure:**

- GitGuardian on PR (existing CI).
- Optional: `go test -count=1 -run 'TestNoHardcodedSecrets' ./...` scanning allowlisted dev defaults.

**Expected result:** No new secret patterns in `*.go` / config templates.

**Actual result:** *Partial — GitGuardian only; dedicated Go test pending.*

**CI evidence:** GitGuardian check on Slice 7 PR (*link when available*).

**Known limitations:** README documents dev passwords for Compose.

---

## S7-PROD-1 — Production configuration gate

**Claim:** A documented production mode refuses to start if required security env vars are missing.

**Test procedure:**

- `go test -count=1 -run 'TestProductionConfig' ./...`

**Expected result:** `TICKET_ENV=production` (or agreed name) without required vars → fatal error before serving.

**Actual result:** *Not recorded yet.*

**CI evidence:** *Pending.*

---

## S7-RATE-1 — Rate limiting (if in Slice 7 scope)

**Claim:** Sustained abuse of `/book` is throttled.

**Test procedure:** *Define when rate limit design is fixed.*

**Expected result:** *TBD.*

**Actual result:** *Not recorded yet.*

**CI evidence:** *Pending.*

---

## Final acceptance (Slice 7)

```
IMPLEMENTATION + AUTOMATED TEST + REPRODUCIBLE RESULT + CI EVIDENCE = PROVEN
```

Checklist:

- [ ] Every row in the claims matrix has **Actual result** filled
- [ ] Every row has a **CI workflow run** reference (not local-only)
- [ ] `go test ./...`, `go test -race ./...`, `go vet ./...` PASS in that run
- [ ] `go test -tags=integration ./integration/...` PASS when security tests need services
- [ ] Booking invariants unchanged (unit + integration booking/persist tests still PASS)
