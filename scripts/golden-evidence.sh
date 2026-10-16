#!/usr/bin/env bash
# Run verification-lab benchmark presets and export JSON for docs/BENCHMARKS.md.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
LAB="${VERILAB_HTTP:-http://127.0.0.1:8090}"
ARCHIVE="${GOLDEN_ARCHIVE:-$ROOT/../cursor/stores/self/internal/golden-benchmarks}"
mkdir -p "$ARCHIVE"
SHA="$(git -C "$ROOT" rev-parse --short HEAD)"
DATE="$(date -u +%Y-%m-%d)"

poll_run() {
  local id="$1" preset="$2" max_wait="${3:-600}"
  local i=0
  while [ "$i" -lt "$max_wait" ]; do
    status="$(curl -sf "$LAB/api/runs/$id" | jq -r '.status')"
    if [ "$status" = "completed" ] || [ "$status" = "failed" ]; then
      curl -sf "$LAB/api/runs/$id/export" -o "$ARCHIVE/${DATE}-${preset}-${SHA}.json"
      echo "exported $preset -> $ARCHIVE/${DATE}-${preset}-${SHA}.json (status=$status)"
      return 0
    fi
    sleep 2
    i=$((i + 2))
  done
  echo "timeout waiting for $preset run $id" >&2
  return 1
}

start_preset() {
  local body="$1"
  curl -sf -X POST "$LAB/api/runs" -H 'Content-Type: application/json' -d "$body" | jq -r '.run_id'
}

echo "golden evidence: lab=$LAB sha=$SHA"

for preset in benchmark-smoke benchmark-saturated-15k benchmark-exhaustion-mini; do
  id="$(start_preset "{\"preset\":\"$preset\"}")"
  poll_run "$id" "$preset" 300
done

id="$(start_preset '{"preset":"benchmark-exhaustion-15k","confirm_heavy":true}')"
poll_run "$id" "benchmark-exhaustion-15k" 900

id="$(start_preset '{"preset":"gates"}')"
poll_run "$id" "gates" 600

id="$(start_preset '{"preset":"failure-lab","confirm_fault":true}')"
poll_run "$id" "failure-lab" 600

echo "done. Summarize key fields from $ARCHIVE/*.json into docs/BENCHMARKS.md"
