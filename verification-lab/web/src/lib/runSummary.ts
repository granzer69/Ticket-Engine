import type { RunReport } from "../types";

const LOAD_PRESETS = new Set([
  "smoke",
  "",
  "heavy-100k",
  "100k",
  "load-1k",
  "1k",
  "load-10k",
  "10k",
  "load-25k",
  "25k",
  "load-50k",
  "50k",
]);

export function isLoadPreset(preset: string): boolean {
  return LOAD_PRESETS.has(preset.toLowerCase());
}

function fmtCount(n: number): string {
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`;
  if (n >= 10_000) return `${Math.round(n / 1000)}k`;
  if (n >= 1_000) return `${(n / 1000).toFixed(1)}k`;
  return String(n);
}

/** Short human line for exhaustion / load runs (200 vs 404 vs errors). */
export function loadRunHumanLine(run: RunReport): string {
  const ok = run.http_200 ?? 0;
  const soldOut = run.http_404 ?? 0;
  const err = run.errors ?? 0;
  const total = run.logical_requests ?? ok + soldOut + err;
  const p = run.preset.toLowerCase();

  if (err > 0) {
    return `load: ${fmtCount(ok)} OK, ${fmtCount(soldOut)} sold-out, ${fmtCount(err)} errors`;
  }
  if (p === "heavy-100k" || p === "100k") {
    if (soldOut > 0) {
      return `exhaustion workload: ${fmtCount(ok)} OK, ${fmtCount(soldOut)} sold-out`;
    }
  }
  if (total > 0 && ok === total) {
    return `all ${fmtCount(total)} requests OK`;
  }
  return `load: ${fmtCount(ok)} OK, ${fmtCount(soldOut)} sold-out`;
}

export function loadRunStatsLine(run: RunReport): string {
  const parts: string[] = [];
  if (run.http_200 != null) parts.push(`200: ${run.http_200}`);
  if (run.http_404 != null) parts.push(`404: ${run.http_404}`);
  if (run.errors != null && run.errors > 0) parts.push(`errors: ${run.errors}`);
  if (run.logical_requests != null) parts.push(`logical: ${run.logical_requests}`);
  if (run.duration_ms != null) parts.push(`${run.duration_ms}ms`);
  if (run.latency_p95_ms != null) parts.push(`p95 ${run.latency_p95_ms}ms`);
  return parts.join(" · ");
}

export function loadRunStatusLabel(run: RunReport): string {
  const base = run.status;
  if (!isLoadPreset(run.preset) || run.logical_requests == null) {
    return base;
  }
  const ok = run.http_200 ?? 0;
  const soldOut = run.http_404 ?? 0;
  const err = run.errors ?? 0;
  const p = run.preset.toLowerCase();

  if (base === "completed") return "completed";
  if (err > 0) return `${base} (errors present)`;
  if ((p === "heavy-100k" || p === "100k") && soldOut > 0 && err === 0) {
    return `${base} (inventory exhausted — ${fmtCount(ok)} OK / ${fmtCount(soldOut)} sold-out)`;
  }
  if (ok < (run.logical_requests ?? 0)) {
    return `${base} (${fmtCount(ok)}/${fmtCount(run.logical_requests ?? 0)} OK)`;
  }
  return base;
}
