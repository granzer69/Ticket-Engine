import { useMemo } from "react";
import { isLoadPreset, loadRunHumanLine, loadRunStatsLine, loadRunStatusLabel } from "../lib/runSummary";
import { useLabStore } from "../store/labStore";

const INV_IDS = ["INV-1", "INV-2", "INV-3", "INV-4", "INV-5", "INV-6", "INV-7"];
const GATE_IDS = ["P1", "P2", "P3", "P4", "P5", "P6", "P7"];

export function ClaimsPanel() {
  const run = useLabStore((s) => s.run);

  const isGates = run?.preset === "gates" || run?.preset === "phase-gates";
  const rowIds = isGates ? GATE_IDS : INV_IDS;

  const rows = useMemo(() => {
    const byId = new Map(run?.claims?.map((c) => [c.id, c]));
    return rowIds.map((id) => {
      const c = byId.get(id);
      const defaultStatus =
        run?.status === "running" && (run.preset === "claims" || isGates) ? "RUNNING" : "NOT RUN";
      return {
        id,
        status: c?.status ?? defaultStatus,
        message: c?.message ?? "",
        failureReason: c?.failure_reason ?? "",
        evidence: c?.evidence,
      };
    });
  }, [run, rowIds, isGates]);

  const loadHeader = run && isLoadPreset(run.preset) && run.logical_requests != null;

  return (
    <section className="panel claims-panel">
      <div className="panel-head">
        <h2>{isGates ? "Phase gates" : "Verification"}</h2>
        {run && (
          <div className="run-head-meta">
            <span className="hint">
              <span className={`status ${run.status}`}>{loadHeader ? loadRunStatusLabel(run) : run.status}</span>
              {" · "}
              {run.preset}
              {run.run_id && <> · {run.run_id.slice(0, 8)}</>}
            </span>
            {loadHeader && (
              <>
                <span className="hint run-stats-line">{loadRunStatsLine(run)}</span>
                <span className="hint run-human-line">{loadRunHumanLine(run)}</span>
              </>
            )}
            {run.extra?.failure_reason && (
              <span className="hint fail-reason">run: {String(run.extra.failure_reason)}</span>
            )}
            {run.error && <span className="hint run-error-inline">{run.error}</span>}
          </div>
        )}
      </div>
      <div className="claims-table">
        {rows.map((r) => (
          <div className="claim-row" key={r.id}>
            <span>{r.id}</span>
            <span>
              <span className={`status ${r.status.replace(" ", "_")}`}>{r.status}</span>
              {r.failureReason && <span className="hint fail-reason">{r.failureReason}</span>}
              {r.message && <span className="msg">{r.message}</span>}
              {r.evidence && Object.keys(r.evidence).length > 0 && (
                <details className="evidence">
                  <summary>evidence</summary>
                  <pre>{JSON.stringify(r.evidence, null, 2)}</pre>
                </details>
              )}
            </span>
          </div>
        ))}
      </div>
      {run && <pre className="export">{JSON.stringify(run, null, 2)}</pre>}
    </section>
  );
}
