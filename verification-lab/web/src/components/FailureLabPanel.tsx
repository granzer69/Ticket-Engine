import { useEffect, useMemo, useState } from "react";
import { pollDeadlineLabel, pollRun } from "../lib/pollRun";
import { useLabStore } from "../store/labStore";
import type { ClaimResult } from "../types";

type ScenarioMeta = {
  id: string;
  title: string;
  warning: string;
  manual_steps?: string[];
  safe_automated: boolean;
  registry_tests: string[];
};

const DEFAULT_IDS = [
  "worker-unavailable",
  "restart-inventory",
  "dlq-growth",
  "reconcile-after-stress",
];

export function FailureLabPanel() {
  const busy = useLabStore((s) => s.busy);
  const setBusy = useLabStore((s) => s.setBusy);
  const setRun = useLabStore((s) => s.setRun);
  const run = useLabStore((s) => s.run);
  const [catalog, setCatalog] = useState<ScenarioMeta[]>([]);

  useEffect(() => {
    fetch("/api/failure-lab/scenarios")
      .then((r) => r.json())
      .then((body) => {
        if (Array.isArray(body)) setCatalog(body as ScenarioMeta[]);
      })
      .catch(() => {
        setCatalog(
          DEFAULT_IDS.map((id) => ({
            id,
            title: id,
            warning: "",
            safe_automated: true,
            registry_tests: [],
          })),
        );
      });
  }, []);

  const isFailureRun = run?.preset === "failure-lab";
  const resultsById = useMemo(() => {
    const m = new Map<string, ClaimResult>();
    run?.claims?.forEach((c) => m.set(c.id, c));
    return m;
  }, [run?.claims]);

  const start = async (scenarios: string[]) => {
    if (
      !window.confirm(
        "Run failure-lab scenarios against the allowlisted target? Destructive faults require manual steps per scenario warnings.",
      )
    ) {
      return;
    }
    setBusy(true);
    setRun(null);
    try {
      const res = await fetch("/api/runs", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          preset: "failure-lab",
          scenarios,
          confirm_fault: true,
        }),
      });
      if (!res.ok) {
        const text = (await res.text()).trim() || `HTTP ${res.status}`;
        setRun({ run_id: "", preset: "failure-lab", status: "failed", error: text });
        return;
      }
      const { run_id } = (await res.json()) as { run_id: string };
      setRun({ run_id, preset: "failure-lab", status: "running" });
      const report = await pollRun(run_id, "failure-lab");
      if (report) setRun(report);
      else {
        setRun({
          run_id,
          preset: "failure-lab",
          status: "failed",
          error: `Polling timed out after ${pollDeadlineLabel("failure-lab")}.`,
        });
      }
    } finally {
      setBusy(false);
    }
  };

  return (
    <section className="panel failure-lab-panel">
      <div className="panel-head">
        <h2>Failure lab</h2>
        {isFailureRun && run && (
          <span className="hint">
            <span className={`status ${run.status}`}>{run.status}</span>
            {run.run_id && <> · {run.run_id.slice(0, 8)}</>}
            {run.extra?.failure_reason && (
              <span className="hint fail-reason"> · {String(run.extra.failure_reason)}</span>
            )}
          </span>
        )}
      </div>
      <p className="muted">
        Safe, bounded probes only (localhost allowlist, server-side Redis). Confirm faults before POST{" "}
        <code>/api/runs</code> preset <code>failure-lab</code>.
      </p>
      <div className="actions">
        <button className="danger" disabled={busy} onClick={() => start([])}>
          Run all scenarios
        </button>
      </div>
      <ul className="failure-scenario-list">
        {catalog.map((s) => {
          const r = resultsById.get(s.id);
          const status =
            r?.status ??
            (isFailureRun && run?.status === "running" ? "RUNNING" : "NOT RUN");
          return (
            <li key={s.id} className="failure-scenario">
              <div className="failure-scenario-head">
                <span className="gate-id">{s.id}</span>
                <span className={`status ${status.replace(" ", "_")}`}>{status}</span>
                <button type="button" className="gate-run" disabled={busy} onClick={() => start([s.id])}>
                  Run
                </button>
              </div>
              <div className="failure-scenario-title">{s.title}</div>
              {s.warning && <p className="warning-text">{s.warning}</p>}
              {s.manual_steps && s.manual_steps.length > 0 && (
                <details className="manual-steps">
                  <summary>Manual checklist</summary>
                  <ol>
                    {s.manual_steps.map((step) => (
                      <li key={step}>{step}</li>
                    ))}
                  </ol>
                </details>
              )}
              {r?.message && <span className="msg">{r.message}</span>}
              {r?.failure_reason && <span className="hint fail-reason">{r.failure_reason}</span>}
              {r?.evidence && Object.keys(r.evidence).length > 0 && (
                <details className="evidence">
                  <summary>evidence</summary>
                  <pre>{JSON.stringify(r.evidence, null, 2)}</pre>
                </details>
              )}
              {s.registry_tests?.length > 0 && (
                <span className="hint muted">tests: {s.registry_tests.join(", ")}</span>
              )}
            </li>
          );
        })}
      </ul>
    </section>
  );
}
