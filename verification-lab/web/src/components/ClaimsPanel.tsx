import { useMemo } from "react";
import { isLoadPreset, loadRunHumanLine, loadRunStatsLine, loadRunStatusLabel } from "../lib/runSummary";
import { useLabStore } from "../store/labStore";

const INV_IDS = ["INV-1", "INV-2", "INV-3", "INV-4", "INV-5", "INV-6", "INV-7"];

export function ClaimsPanel() {
  const run = useLabStore((s) => s.run);

  const rows = useMemo(() => {
    const byId = new Map(run?.claims?.map((c) => [c.id, c]));
    return INV_IDS.map((id) => {
      const c = byId.get(id);
      return {
        id,
        status: c?.status ?? (run?.preset === "claims" ? "NOT RUN" : "NOT RUN"),
        message: c?.message ?? "",
      };
    });
  }, [run]);

  const loadHeader = run && isLoadPreset(run.preset) && run.logical_requests != null;

  return (
    <section className="panel claims-panel">
      <div className="panel-head">
        <h2>Verification</h2>
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
              {r.message && <span className="msg">{r.message}</span>}
            </span>
          </div>
        ))}
      </div>
      {run && <pre className="export">{JSON.stringify(run, null, 2)}</pre>}
    </section>
  );
}
