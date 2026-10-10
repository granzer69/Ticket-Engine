import { useCallback } from "react";
import { pollDeadlineLabel, pollRun } from "../lib/pollRun";
import { useLabStore } from "../store/labStore";

const GATE_IDS = ["P1", "P2", "P3", "P4", "P5", "P6", "P7"];

export function RunControl() {
  const busy = useLabStore((s) => s.busy);
  const setBusy = useLabStore((s) => s.setBusy);
  const setRun = useLabStore((s) => s.setRun);
  const run = useLabStore((s) => s.run);
  const clearChart = useLabStore((s) => s.clearChart);
  const pushRunSnaps = useLabStore((s) => s.pushChartFromRunSnapshots);

  const gateById = (id: string) => run?.claims?.find((c) => c.id === id);

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

  const confirmBenchmarkExhaustion = () => {
    if (
      !window.confirm(
        "Run benchmark-exhaustion-15k (100k logical attempts; assumes 15k ticket seed)?",
      )
    ) {
      return;
    }
    start({ preset: "benchmark-exhaustion-15k", confirm_heavy: true });
  };

  const isGatesRun = run?.preset === "gates" || run?.preset === "phase-gates";

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
        <button disabled={busy} onClick={() => start({ preset: "gates" })}>
          Gates P1..P7
        </button>
        <button className="danger" disabled={busy} onClick={confirmHeavy}>
          Heavy · 100k
        </button>
      </div>
      <div className="actions benchmark-actions">
        <span className="hint muted">Benchmarks (15k seed assumption in export)</span>
        <button disabled={busy} onClick={() => start({ preset: "benchmark-smoke" })}>
          Bench smoke · 100
        </button>
        <button disabled={busy} onClick={() => start({ preset: "benchmark-exhaustion-mini" })}>
          Bench exhaust mini · 5k
        </button>
        <button disabled={busy} onClick={() => start({ preset: "benchmark-saturated-15k" })}>
          Bench saturated · 15k
        </button>
        <button className="danger" disabled={busy} onClick={confirmBenchmarkExhaustion}>
          Bench exhaustion · 100k
        </button>
      </div>
      <div className="phase-gates">
        <h3>Phase gates (P1..P7)</h3>
        <p className="muted">Runs go test subprocesses + live probes (readyz, API key) against VERILAB_TARGET.</p>
        <ul className="gate-list">
          {GATE_IDS.map((p) => {
            const g = gateById(p);
            const status = g?.status ?? (isGatesRun && run?.status === "running" ? "RUNNING" : "NOT RUN");
            const ranAt = g?.evidence?.ran_at;
            return (
              <li key={p}>
                <span className="gate-id">{p}</span>
                <span className={`status ${status.replace(" ", "_")}`}>{status}</span>
                <button
                  type="button"
                  className="gate-run"
                  disabled={busy}
                  onClick={() => start({ preset: "gates", gates: [p] })}
                >
                  Run
                </button>
                {g?.failure_reason && <span className="hint fail-reason">{g.failure_reason}</span>}
                {g?.message && <span className="msg">{g.message}</span>}
                {ranAt != null && <span className="hint">{String(ranAt)}</span>}
              </li>
            );
          })}
        </ul>
      </div>
    </section>
  );
}
