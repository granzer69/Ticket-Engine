import { useCallback } from "react";
import { pollDeadlineLabel, pollRun } from "../lib/pollRun";
import { useLabStore } from "../store/labStore";
export function RunControl() {
  const busy = useLabStore((s) => s.busy);
  const setBusy = useLabStore((s) => s.setBusy);
  const setRun = useLabStore((s) => s.setRun);
  const run = useLabStore((s) => s.run);
  const clearChart = useLabStore((s) => s.clearChart);
  const pushRunSnaps = useLabStore((s) => s.pushChartFromRunSnapshots);

  const start = useCallback(
    async (body: Record<string, unknown>) => {
      const preset = String(body.preset ?? "smoke");
      setBusy(true);
      setRun(null);
      clearChart();
      try {
        const res = await fetch("/api/runs", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(body),
        });
        if (!res.ok) {
          const text = (await res.text()).trim() || `HTTP ${res.status}`;
          setRun({
            run_id: "",
            preset,
            status: "failed",
            error: text,
          });
          return;
        }
        const { run_id } = (await res.json()) as { run_id: string };
        setRun({ run_id, preset, status: "running" });
        const report = await pollRun(run_id, preset);
        if (report) {
          setRun(report);
          if (report.snapshot_start && report.snapshot_end) {
            pushRunSnaps(report.snapshot_start, report.snapshot_end);
          }
        } else {
          setRun({
            run_id,
            preset,
            status: "failed",
            error: `Polling timed out after ${pollDeadlineLabel(preset)}; the run may still be executing on the server.`,
          });
        }
      } finally {
        setBusy(false);
      }
    },
    [clearChart, pushRunSnaps, setBusy, setRun],
  );

  const confirmHeavy = () => {
    if (!window.confirm("Run 100,000 logical booking requests against the allowlisted target?")) return;
    start({ preset: "heavy-100k", confirm_heavy: true });
  };

  return (
    <section className="panel run-panel">
      <div className="panel-head">
        <h2>Run control</h2>
        {busy && run?.preset && (
          <span className="hint muted">Polling up to {pollDeadlineLabel(run.preset)}…</span>
        )}
      </div>
      {run?.error && (
        <div className="run-error" role="alert">
          {run.error}
        </div>
      )}
      <div className="actions">
        <button className="primary" disabled={busy} onClick={() => start({ preset: "smoke" })}>
          Smoke · 100
        </button>
        <button disabled={busy} onClick={() => start({ preset: "claims", claims: ["INV-1", "INV-2", "INV-3"] })}>
          INV-1..3
        </button>
        <button disabled={busy} onClick={() => start({ preset: "claims" })}>
          INV-1..7
        </button>
        <button className="danger" disabled={busy} onClick={confirmHeavy}>
          Heavy · 100k
        </button>
      </div>
      <div className="phase-gates">
        <h3>Phase gates (P1..P7)</h3>
        <p className="muted">NOT RUN — scenarios not wired in lab runner (see claim registry).</p>
        <ul>
          {["P1", "P2", "P3", "P4", "P5", "P6", "P7"].map((p) => (
            <li key={p}>
              <span>{p}</span> <span className="status NOT_RUN">NOT RUN</span>
            </li>
          ))}
        </ul>
      </div>
    </section>
  );
}
